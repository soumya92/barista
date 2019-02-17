# Copyright 2019 Google Inc.
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

require_relative '../rb/gofile.rb'

Unit = Struct.new(:type, :base, :unit)
UNITS = [
  Unit.new('Acceleration', 'MetersPerSecondSquared', 'm/s²'),
  Unit.new('Angle', 'Radians', 'rad'),
  Unit.new('Area', 'SquareMeters', 'm²'),
  Unit.new('Datarate', 'BytesPerSecond', 'B/s'),
  Unit.new('Datasize', 'Bytes', 'B'),
  Unit.new('ElectricCurrent', 'Amperes', 'A'),
  Unit.new('Energy', 'Joules', 'J'),
  Unit.new('Force', 'Newtons', 'N'),
  Unit.new('Frequency', 'Hertz', 'Hz'),
  Unit.new('Length', 'Meters', 'm'),
  Unit.new('Mass', 'Grams', 'g'),
  Unit.new('Power', 'Watts', 'W'),
  Unit.new('Pressure', 'Pascals', 'Pa'),
  Unit.new('Speed', 'MetersPerSecond', 'm/s'),
  Unit.new('Voltage', 'Volts', 'V'),
  Unit.new('Volume', 'CubicMeters', 'm³'),
  Unit.new('AmountOfSubstance', 'Moles', 'mol'),
  Unit.new('ElectricalConductance', 'Siemens', 'S'),
  Unit.new('ElectricalResistance', 'Ohms', 'Ω'),
  Unit.new('Illuminance', 'Lux', 'lx'),
  Unit.new('LuminousFlux', 'Lumen', 'lm'),
  Unit.new('LuminousIntensity', 'Candela', 'cd')
].freeze

write_go_file('siunit.go') do |out|
  out.write <<~HEADER
    package format

    import "github.com/martinlindhe/unit"

    // SIUnit formats a unit.Unit value to an appropriately scaled base unit.
    // For example, SIUnit(length) is equivalent to SI(length.Meters(), "m").
    // For non-base units (e.g. feet), use SI(length.Feet(), "ft").
    func SIUnit(val interface{}) (Value, bool) {
    \tswitch v := val.(type) {
  HEADER

  UNITS.each do |u|
    out.write <<~CASE
      \tcase unit.#{u.type}:
      \t\treturn SI(v.#{u.base}(), "#{u.unit}"), true
    CASE
  end

  out.write <<~FOOTER
    \t}
    \treturn Value{}, false
    }
  FOOTER
end

write_go_file('siunit_test.go') do |out|
  out.write <<~HEADER
    package format

    import "github.com/martinlindhe/unit"

    // An example for each unit that can be handled by the SIUint function. This
    // is used to test that all declared units in the unit package are handled.
    var siUnitsHandled = map[string]interface{}{
  HEADER
  UNITS.each do |u|
    out.puts "\t\"#{u.type}\": unit.#{u.type}(1),"
  end
  out.puts '}'
end
