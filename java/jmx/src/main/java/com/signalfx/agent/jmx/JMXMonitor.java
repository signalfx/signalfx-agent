package com.signalfx.agent.jmx;

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


public class JMXMonitor implements SignalFxMonitor<JMXMonitor.JMXConfig> {
    private static Logger logger = Logger.getLogger(JMXMonitor.class.getName());

    private final Timer timer = new Timer();

    public static class JMXConfig extends MonitorConfig {
        public String serviceURL;
        public String groovyScript;
        public String username;
        public String password;
    }

    public void configure(JMXConfig conf, AgentOutput output) {
        Client client;
        try {
            client = new Client(conf.serviceURL, conf.username, conf.password);
        } catch(MalformedURLException e) {
            throw new ConfigureError("Malformed serviceUrl: ", e);
        }

        GroovyRunner runner = new GroovyRunner(conf.groovyScript, output, client);

        timer.scheduleAtFixedRate(MonitorUtil.wrapTimerTask(() -> {
            try {
                runner.run();
            } catch (Throwable e) {
                logger.log(Level.SEVERE, "Error gathering JMX metrics", e);
            }

        }), 0, conf.intervalSeconds * 1000);
    }

    public void shutdown() {
        timer.cancel();
        return;
    }

    public static void main(String[] args) {
        SignalFxMonitorRunner runner = new SignalFxMonitorRunner(new JMXMonitor(), JMXConfig.class);
        runner.run();
    }
}
