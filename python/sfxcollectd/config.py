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
            if v is None:
                logging.debug("dropping configuration %s because its value is None", k)
                continue
            if isinstance(v, (tuple, list)):
                if len(v) == 0:
                    logging.debug("dropping configuration %s because its value is an empty list or tuple", k)
                    continue
                values = v
            elif isinstance(v, (str, unicode)):
                if v == "":
                    logging.debug("dropping configuration %s because its value is an empty string", k)
                    continue
                values = (v,)
            elif isinstance(v, (int, bool)):
                values = (v,)
            elif isinstance(v, dict):
                if len(v) == 0 :
                    logging.debug("dropping configuration %s because its value is an empty dictionary", k)
                    continue
                if "#flatten" in v and "values" in v:
                    conf.children += [cls(root=conf, key=k, values=item, children=[]) for item in v.get("values") or [] if item is not None]
                    continue
                dict_conf = cls.from_monitor_config(v)
                children = dict_conf.children
                values = dict_conf.values
            else:
                logging.error("Cannot convert monitor config to collectd config: %s: %s", k, v)

            conf.children.append(cls(root=conf, key=k, values=values, children=children))

        return conf
