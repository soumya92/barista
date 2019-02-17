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

# This program generates a go interface for C calls by wrapping each used type
# and function, a real implementation of the interface, and a mock
# implementation of the interface for testing.
# This is required because C types cannot be used in tests, making it very
# difficult to mock C calls for tests. Traditional techniques, such as
# `var timeNow = time.Now` do not work with C.
#
# It works by dumping all function prototypes reachable from the given .h file
# and filtering the list to only the functions used in the given .go files.
#
# It requires gcc and universal ctags to be installed and in $PATH.
# pkg-config is only required if pkg-config packages are provided.

require_relative 'funcs.rb'
require_relative 'generators.rb'
require_relative '../../rb/run_cmd.rb'

def headers(header_filename, pkg_config: nil)
  include_dirs = ['.']
  include_dirs += run_cmd('gcc -Wp,-v -x c /dev/null -fsyntax-only')
                  .lines.select { |l| l.start_with? ' ' }.map(&:strip)
  unless pkg_config.nil?
    include_dirs += run_cmd('pkg-config', pkg_config, '--cflags-only-I')
                    .split(/\s*-I\s*/).map(&:strip).reject(&:empty?)
  end

  header_file = include_dirs.map { |path| File.join(path, header_filename) }
                            .find { |file| File.file?(file) }
  abort "Could not find #{header_filename} in #{include_dirs}" if header_file.nil?

  run_cmd('gcc', '-M', header_file).sub(/[^:]+\.o: /, '').gsub(/\\\n/m, '').split
end

def make_capi(header_filename, go_src_files:, pkg_config: nil, library: nil, package: nil)
  go_code = go_src_files.map { |f| run_cmd('gcc', '-fpreprocessed', '-dD', '-E', '-x', 'c', '-P', f) }
                        .join("\n")

  library ||= File.basename(header_filename, '.h')
  package ||= go_code.lines
                     .select { |line| line.start_with? 'package ' }
                     .map { |pkg_decl| pkg_decl.delete_prefix 'package ' }
                     .first
  abort "No package specified or found in go files #{go_src_files}" if package.nil?

  args = ['-x', '--c-kinds=p']
  args << "--_xformat=%N\t%{typeref}\t%{signature}" # rubocop:disable Style/FormatStringToken
  args += headers(header_filename, pkg_config: pkg_config)
  c_functions = run_cmd('ctags', *args)
                .lines.map { |entry| Prototype.new(*entry.strip.split("\t")) }
                .select { |func| go_code.include? "#{library}.#{func.name}" }

  abort "No C function usage detected! Are calls prefixed with `#{library}.`?" if c_functions.empty?

  write_capi_file(library, package, pkg_config, header_filename, c_functions)
  write_test_file(library, package, c_functions)
end
