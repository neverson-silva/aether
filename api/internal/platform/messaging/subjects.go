package messaging

const (
	JobsPrefix         = "aether.jobs."
	EventsPrefix       = "aether.events."
	LivePrefix         = "aether.live."
	DLQPrefix          = "aether.dlq."
	StatePrefix        = "aether.state."
	MonitoringPrefix   = "aether.monitoring."
	NotifyPrefix       = "aether.notify."
	MonitoringSnapshot = MonitoringPrefix + "snapshot"
)

func Jobs(stream string) string     { return JobsPrefix + stream }
func Events(topic string) string    { return EventsPrefix + topic }
func Live(topic string) string      { return LivePrefix + topic }
func DLQ(stream string) string      { return DLQPrefix + stream }
func State(key string) string       { return StatePrefix + key }
func NotifyOrg(orgID string) string { return NotifyPrefix + "org." + orgID }
