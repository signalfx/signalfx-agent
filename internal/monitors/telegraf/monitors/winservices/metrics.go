package winservices

// GAUGE(win_services.state): The state of the windows service.  Possible values
// are: `1` (Stopped), `2` (Start Pending), `3` (Stop Pending), `4` (Running),
// `5` (Continue Pending), `6` (Pause Pending), and `7` (Paused).

// GAUGE(win_services.startup_mode): The configured start up mode of the window
// windows service.  Possible values are: `0` (Boot Start), `1` (System Start),
// `2` (Auto Start), `3` (Demand Start), `4` (disabled).
