"""
Logic for converting from the agent config format to the Collectd-python config
object format
"""

import logging

logger = logging.getLogger(__name__)


class Config(object):
    """
    Dummy class that we use to put config that conforms to the collectd-python
    Config class

    See https://collectd.org/documentation/manpages/collectd-python.5.shtml#config
    """

    def __init__(self, root=None, key=None, values=None, children=None):
        self.root = root
        self.key = key
        self.values = values
        self.children = children

    @classmethod
    def from_monitor_config(cls, monitor_plugin_config):
        """
        Converts config as expressed in the monitor to the Collectd Config
        interface.
        """
        assert isinstance(monitor_plugin_config, dict)

        conf = cls(root=None)
        conf.children = []
        for k, v in monitor_plugin_config.items():
            values = None
            children = None
            if isinstance(v, (tuple, list)):
                values = v
            elif isinstance(v, (int, str, unicode)):
                values = (v,)
            elif isinstance(v, dict):
                dict_conf = cls.from_monitor_config(v)
                children = dict_conf.children
                values = dict_conf.values
            else:
                logging.error("Cannot convert monitor config to collectd config: %s: %s", k, v)

            conf.children.append(cls(root=conf, key=k, values=values, children=children))

        return conf
