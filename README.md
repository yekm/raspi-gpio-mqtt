# sysfs-gpio-mqtt

This is a fork of heathbar/raspi-gpio-mqtt with support for both direction of gpio pins
and auto-discovery in home assistant.

After launching with `-p 19:binary_sensor:somebutton,13:switch:somelights`
you will see in homeassistant a switch named `somelights` and a sensor named `somebutton`.
No additional configuration needed, except enabled autodiscovery in homeassistant.
