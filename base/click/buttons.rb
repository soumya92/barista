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

require_relative '../../rb/gofile.rb'

SCROLLS = %w[Left Right Up Down].freeze
BUTTONS = %w[Left Right Middle Back Forward].freeze

method_and_btn = BUTTONS.map { |n| [n, "Button#{n}"] } +
                 SCROLLS.map { |n| ["Scroll#{n}", "Scroll#{n}"] }

write_go_file('buttons.go') do |out|
  out.puts <<~HEADER
    package click

    import "barista.run/bar"
  HEADER

  method_and_btn.each do |method, btn|
    out.write(<<~METHOD)

      // #{method} creates a click handler that invokes the given function
      // when a #{btn} event is received.
      func #{method}(do func()) func(bar.Event) {
      \treturn #{method}E(DiscardEvent(do))
      }

      // #{method}E wraps the click handler so that it is only triggered by a
      // #{btn} event.
      func #{method}E(handler func(bar.Event)) func(bar.Event) {
      \treturn ButtonE(handler, bar.#{btn})
      }
    METHOD
  end
  method_and_btn.each do |method, btn|
    out.write(<<~METHOD)

      // #{method} invokes the given function on #{btn} events.
      func (m Map) #{method}(do func()) Map {
      \treturn m.#{method}E(DiscardEvent(do))
      }

      // #{method}E sets the click handler for #{btn} events.
      func (m Map) #{method}E(handler func(bar.Event)) Map {
      \treturn m.Set(bar.#{btn}, handler)
      }
    METHOD
  end
end
