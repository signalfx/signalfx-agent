package com.signalfx.agent.testmonitor;

import java.net.MalformedURLException;
import java.util.Timer;
import java.util.logging.Level;
import java.util.logging.Logger;

import com.signalfx.agent.AgentOutput;
import com.signalfx.agent.ConfigureError;
import com.signalfx.agent.MonitorConfig;
import com.signalfx.agent.SignalFxMonitor;
import com.signalfx.agent.SignalFxMonitorRunner;
import com.signalfx.agent.MonitorUtil;


public class TestMonitor implements SignalFxMonitor<TestMonitor.TestConfig> {
    private static Logger logger = Logger.getLogger(TestMonitor.class.getName());

    private final Timer timer = new Timer();

    public static class TestConfig extends MonitorConfig {
        public String a;
    }

    public void configure(TestConfig conf, AgentOutput output) {
        timer.scheduleAtFixedRate(MonitorUtil.wrapTimerTask(() -> {
			output.sendDatapoint(MonitorUtil.makeGauge("my.gauge", 1, MonitorUtil.newDims("a", conf.a)));
        }), 0, conf.intervalSeconds * 1000);
    }

    public void shutdown() {
        timer.cancel();
        return;
    }

    public static void main(String[] args) {
        SignalFxMonitorRunner runner = new SignalFxMonitorRunner(new TestMonitor(), TestConfig.class);
        runner.run();
    }
}
