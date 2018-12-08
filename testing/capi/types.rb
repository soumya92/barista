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

C_TO_GO_TYPES = {
  'unsigned' => 'uint32',
  'int' => 'int32',
  'char' => 'int8',
  'short' => 'int16',
  'long' => 'int64',
  'float' => 'float32',
  'double' => 'float64',
  'long long' => 'int64',
  'long double' => 'float64'
}.freeze

GO_KEYWORDS = %w[
  break default func interface select case defer go map struct chan else goto
  package switch const fallthrough if range type continue for import return var
].freeze

def from_c_type(type)
  type = type.gsub('__extension__', '')
             .gsub('__restrict', '')
             .gsub('__BEGIN_DECLS', '')
             .gsub(/\bconst\b/, '')
             .strip
             .sub(/^struct /, 'struct_')
             .sub(/^enum /, 'enum_')
             .delete_suffix(' int')
  if C_TO_GO_TYPES.key?(type.delete_prefix('unsigned '))
    CTypePrim.new(type)
  elsif type == 'char *'
    CTypeString.new
  elsif type == 'void *'
    CTypeVoidPtr.new
  elsif type[-1] == '*'
    CTypePtr.new(type)
  elsif type == 'void'
    CTypeVoid.new
  else
    CType.new("ctyp_#{type}", "C.#{type}")
  end
end

# Generic type, usually a C struct or other simple C type without a simple
# equivalent in go.
class CType
  def initialize(gotype, ctype, aliases = { ctype => gotype })
    @gotype = gotype
    @ctype = ctype
    @aliases = aliases
  end

  attr_accessor :gotype, :ctype, :aliases

  def argument(expr, tmpvar)
    [tmpvar, "#{tmpvar} := #{@ctype}(#{expr})"]
  end

  def return(expr, tmpvar)
    "#{tmpvar} := #{expr}\nreturn #{@gotype}(#{tmpvar})"
  end
end

# Void type.
class CTypeVoid < CType
  def initialize
    super(nil, nil, {})
  end

  def argument(_expr, _tmpvar)
    raise 'Trying to pass void argument to C function!'
  end

  def return(expr, _tmpvar)
    expr.to_s
  end
end

# char* mapped to string.
class CTypeString < CType
  def initialize
    super('string', 'string', {})
  end

  def argument(expr, tmpvar)
    [tmpvar, "#{tmpvar} := C.CString(#{expr})\ndefer C.free(unsafe.Pointer(#{tmpvar}))"]
  end

  def return(expr, tmpvar)
    "#{tmpvar} := #{expr}\nreturn C.GoString(#{tmpvar})"
  end
end

# Generic pointer (void*)
class CTypeVoidPtr < CType
  def initialize
    super('unsafe.Pointer', 'unsafe.Pointer', {})
  end

  def argument(expr, _tmpvar)
    [expr, nil]
  end

  def return(expr, _tmpvar)
    "return #{expr}"
  end
end

# A simple C type with a corresponding equivalent in go, e.g. int, long, double.
class CTypePrim < CType
  def initialize(type)
    unsigned = type.start_with? 'unsigned '
    ctype = type.delete_prefix 'unsigned '
    gotype = C_TO_GO_TYPES[ctype]
    gotype = "u#{gotype}" if unsigned
    super(gotype, "C.#{ctype.delete ' '}", {})
  end
end

# A fixed-size array of a simple C type.
class CTypePrimAry < CType
  def initialize(type, count)
    super("[#{count}]#{type.gotype}", "[#{count}]#{type.ctype}", type.aliases)
  end

  def argument(expr, tmpvar)
    [tmpvar, "#{tmpvar} := (#{@ctype})(#{expr})"]
  end

  def return(expr, tmpvar)
    "#{tmpvar} := #{expr}\nreturn (#{gotype})(#{tmpvar})"
  end
end

# A pointer to another type.
class CTypePtr < CType
  def initialize(type)
    pointee = from_c_type(type[0...-1].strip)
    super("*#{pointee.gotype}", "*#{pointee.ctype}", pointee.aliases)
  end

  def argument(expr, tmpvar)
    [tmpvar, "#{tmpvar} := (#{@ctype})(#{expr})"]
  end

  def return(expr, tmpvar)
    "#{tmpvar} := #{expr}\nreturn (#{gotype})(#{tmpvar})"
  end
end
