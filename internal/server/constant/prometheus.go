package constant

const (
	MetricEcoFlowMqttEnabled       = "ecolinker_ecoflow_mqtt_enabled"
	MetricEcoFlowMqttEnabledHelp   = "EcoFlow MQTT is enabled, 0 indicating not enabled, 1 indicating that it's enabled"
	MetricEcoFlowMqttConnected     = "ecolinker_ecoflow_mqtt_connected"
	MetricEcoFlowMqttConnectedHelp = "EcoFlow MQTT is connected, 0 indicating not connected, 1 indicating that it's connected"

	MetricMqttForwardEnabled       = "ecolinker_mqtt_forward_enabled"
	MetricMqttForwardEnabledHelp   = "MQTT forward is enabled, 0 indicating not enabled, 1 indicating that it's enabled"
	MetricMqttForwardConnected     = "ecolinker_mqtt_forward_connected"
	MetricMqttForwardConnectedHelp = "MQTT forward is connected, 0 indicating not connected, 1 indicating that it's connected"

	MetricEcoFlowMqttMessagesReceived     = "ecolinker_ecoflow_mqtt_messages_received"
	MetricEcoFlowMqttMessagesReceivedHelp = "Messages received from EcoFlow MQTT"

	MetricEcoFlowMqttMessageLastReceived     = "ecolinker_ecoflow_mqtt_message_last_received"
	MetricEcoFlowMqttMessageLastReceivedHelp = "Last received message timestamp (UNIX epoch in seconds) from EcoFlow MQTT"

	MetricEcoFlowMqttMessageLastReceivedWithPayload     = "ecolinker_ecoflow_mqtt_message_last_received_with_payload"
	MetricEcoFlowMqttMessageLastReceivedWithPayloadHelp = "Last received message timestamp (UNIX epoch in seconds) from EcoFlow MQTT where payload had data"

	MetricCollectorInvocations     = "ecolinker_collector_invocations"
	MetricCollectorInvocationsHelp = "Invocations of a collector"

	MetricCollectorInvocationLast     = "ecolinker_collector_last_invocation"
	MetricCollectorInvocationLastHelp = "Last invocation timestamp (UNIX epoch in seconds) for a collector"
)
