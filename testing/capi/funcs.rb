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

require_relative 'types.rb'

Argument = Struct.new(:name, :type) do
  def initialize(sig, idx)
    type, _, name = sig.rpartition ' '
    if type.empty? || (name == '*')
      type = "#{type} #{name}".strip
      name = "unnamed_#{idx}"
    else
      name = "arg_#{name}"
    end
    if name[-1] == ']'
      name, _, count = name[0...-1].rpartition '['
      super(name, CTypePrimAry.new(from_c_type(type), count))
      return
    end
    super(name, from_c_type(type))
  end
end

def parse_args(signature)
  in_fn_ptr = 0
  curr = ''
  args = []
  signature.each_char do |c|
    if c == '('
      in_fn_ptr += 1
      next
    end
    if in_fn_ptr.positive?
      in_fn_ptr -= 1 if c == ')'
      curr = "void * funcptr_#{args.size}" if in_fn_ptr.zero?
      next
    end
    if c == ','
      args << curr unless curr.empty?
      curr = ''
      next
    end
    curr += c
  end
  args << curr
  args.reject { |arg| arg.strip == 'void' || arg.strip == '...' }
      .map.with_index { |arg, idx| Argument.new(arg, idx) }
end

Prototype = Struct.new(:name, :type, :args) do
  def initialize(name, typeref, signature)
    name = "_#{name}" if GO_KEYWORDS.include? name
    super(name,
      from_c_type(typeref.sub('typename:', '').sub('struct:', 'struct ')),
      parse_args(signature[1...-1]))
  end

  def go_func
    arglist = args.map { |a| "#{a.name} #{a.type.gotype}" }
    "#{name}(#{arglist.join(', ')}) #{type.gotype}"
  end

  def go_type
    argtypes = args.map { |a| a.type.gotype.to_s }
    "func(#{argtypes.join(', ')}) #{type.gotype}"
  end
end
