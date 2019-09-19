package com.signalfx.agent.jmx;

import java.io.IOException;
import java.net.MalformedURLException;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;
import java.util.logging.Level;
import java.util.logging.Logger;

import javax.management.MBeanServerConnection;
import javax.management.ObjectName;
import javax.management.remote.JMXConnector;
import javax.management.remote.JMXConnectorFactory;
import javax.management.remote.JMXServiceURL;

public class Client {
    private static Logger logger = Logger.getLogger(Client.class.getName());

    private final JMXServiceURL url;
    private final String username;
    private final String password;
    private JMXConnector jmxConn;

    Client(String serviceUrl, String username, String password) throws MalformedURLException {
        this.url = new JMXServiceURL(serviceUrl);
        this.username = username;
        this.password = password;
    }

    private MBeanServerConnection ensureConnected() {
        if (jmxConn != null) {
            try {
                return jmxConn.getMBeanServerConnection();
            } catch (IOException e) {
                // Go on and reestablish the connection below if this is reached.
            }
        }

        try {
            Map<String,Object> env = null;
            if (username != null && !username.equals("")) {
                env = new HashMap();
                env.put(JMXConnector.CREDENTIALS, new String[]{this.username, this.password});
            }
            jmxConn = JMXConnectorFactory.connect(url, env);
            return jmxConn.getMBeanServerConnection();
        } catch (IOException e) {
            logger.log(Level.WARNING, "Could not connect to remote JMX server: ", e);
            return null;
        }
    }

    public MBeanServerConnection getConnection() {
        return ensureConnected();
    }

    public Set<ObjectName> query(ObjectName objectName) {
        MBeanServerConnection mbsc = ensureConnected();
        if (mbsc == null) {
            return null;
        }

        try {
             return mbsc.queryNames(objectName, null);
        } catch (IOException e) {
            jmxConn = null;
            return null;
        }
    }
}
