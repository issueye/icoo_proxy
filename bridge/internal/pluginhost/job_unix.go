//go:build unix

package pluginhost

// Unix uses process groups (Setpgid); no Job Object equivalent needed.
type jobHolder struct{}

func newKillOnCloseJob() (*jobHolder, error) { return &jobHolder{}, nil }

func (j *jobHolder) close() {}

func attachProcessToJob(job *jobHolder, pid int) error { return nil }
