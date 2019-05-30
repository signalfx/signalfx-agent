from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf


def get_metric_type(index):
    for metric_type, idx in sf_pbuf.MetricType.items():
        if int(index) == idx:
            return metric_type
    return str(index)


def print_dp_or_event(dp_or_event):
    """
    Prints datapoint object in the format:
    <metric_name> (<metric_type>) [<dim_key>=<dim_value>; ...] = <metric_value> (<timestamp_in_seconds>)
     - <property_key> = <property_value>
     - ...

    Prints event object in the format:
    <event_type> (<event_category>) [<dim_key>=<dim_value>; ...] (<timestamp_in_seconds>)
     - <property_key> = <property_value>
     - ...
    """

    def _get_event_category(number):
        for category, num in sf_pbuf.EventCategory.items():
            if int(number) == num:
                return category
        return str(number)

    def _get_pretty_dims(dims):
        str_dims = ""
        if dims:
            str_dims = " ["
            for dim in sorted(dims, key=lambda dim: dim.key):
                str_dims += dim.key + "=" + dim.value + "; "
            str_dims = str_dims.rstrip("; ") + "]"
        return str_dims

    def _get_pretty_value(value):
        return str(value).split(":", 1)[-1].strip()

    def _get_pretty_props(props):
        str_props = ""
        for prop in sorted(props, key=lambda prop: prop.key):
            str_props += "\n - " + prop.key + " = " + _get_pretty_value(prop.value)
        return str_props

    assert isinstance(dp_or_event, (sf_pbuf.DataPoint, sf_pbuf.Event)), "unsupported type '%s'!" % type(dp_or_event)
    str_dp_or_event = ""
    if isinstance(dp_or_event, sf_pbuf.DataPoint):
        str_dp_or_event = dp_or_event.metric
        str_dp_or_event += " (" + get_metric_type(dp_or_event.metricType) + ")"
        str_dp_or_event += _get_pretty_dims(dp_or_event.dimensions)
        str_dp_or_event += " = " + _get_pretty_value(dp_or_event.value)
    else:
        str_dp_or_event = dp_or_event.eventType
        str_dp_or_event += " (" + _get_event_category(dp_or_event.category) + ")"
        str_dp_or_event += _get_pretty_dims(dp_or_event.dimensions)
    str_dp_or_event += " (" + str(int(dp_or_event.timestamp) / 1000) + ")"
    if hasattr(dp_or_event, "properties"):
        str_dp_or_event += _get_pretty_props(dp_or_event.properties)
    print(str_dp_or_event)
