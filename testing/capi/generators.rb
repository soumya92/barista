# Copyright 2018 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

require 'set'

require_relative 'gofile.rb'

def write_capi_file(library, package, pkg_config, header_filename, c_functions)
  aliases = Set.new(
    c_functions.flat_map { |fn| [fn.type] + fn.args.map(&:type) }
  ).map(&:aliases).reduce(&:merge)

  write_go_file("#{library}_capi.go") do |out|
    out.puts "package #{package}"
    out.puts "// #cgo pkg-config: #{pkg_config}" unless pkg_config.nil?
    out.puts(<<~PREAMBLE)
      // #include <#{header_filename}>
      import "C"
      import "unsafe"

      #{aliases.map { |ctype, gotype| "type #{gotype} = #{ctype}" }.join "\n"}

      type #{library}I interface {
        #{c_functions.map(&:go_func).join "\n"}
      }

      type #{library}Impl struct {}

      var #{library} #{library}I = #{library}Impl{}
    PREAMBLE
    c_functions.each do |fn|
      out.puts "func (#{library}Impl) #{fn.go_func} {"
      c_args = []
      fn.args.each do |arg|
        c_arg, stmt = arg.type.argument arg.name, "tmp_#{arg.name}"
        out.puts stmt unless stmt.nil?
        c_args << c_arg
      end
      expr = "C.#{fn.name}(#{c_args.join ', '})"
      out.puts fn.type.return(expr, 'result_c')
      out.puts '}'
    end
  end
end

def write_test_file(library, package, c_functions)
  write_go_file("#{library}_capi_for_test.go") do |out|
    out.puts "package #{package}"
    out.puts(<<~PREAMBLE)
      import "sync"

      type #{library}Tester struct {
        mu sync.Mutex
    PREAMBLE
    c_functions.each do |fn|
      out.puts "mock_#{fn.name} #{fn.go_type}"
    end
    out.puts(<<~PREAMBLE)
      }

      func #{library}Test() *#{library}Tester {
        tester := &#{library}Tester{}
        #{library} = tester
        return tester
      }
    PREAMBLE
    c_functions.each do |fn|
      out.puts(<<~MOCK)

        func (t *#{library}Tester) on_#{fn.name}(fn #{fn.go_type}) *#{library}Tester {
          t.mu.Lock()
          defer t.mu.Unlock()
          t.mock_#{fn.name} = fn
          return t
        }
      MOCK
      out.puts(<<~FUNC)

        func (t *#{library}Tester) #{fn.go_func} {
          t.mu.Lock()
          defer t.mu.Unlock()
      FUNC
      if fn.type.gotype.nil?
        out.puts(<<~NILCALL)
            if t.mock_#{fn.name} != nil {
              t.mock_#{fn.name}(#{fn.args.map(&:name).join ', '})
            }
          }
        NILCALL
      else
        out.puts(<<~CALL)
            var ret #{fn.type.gotype}
            if t.mock_#{fn.name} != nil {
              ret = t.mock_#{fn.name}(#{fn.args.map(&:name).join ', '})
            }
            return ret
          }
        CALL
      end
    end
  end
end
