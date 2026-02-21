// Package mutation is a test fixture for mutation analysis.
package mutation

// Counter demonstrates pointer receiver mutation.
type Counter struct {
	count int
	name  string
}

// Increment mutates receiver field 'count'.
func (c *Counter) Increment() {
	c.count++
}

// SetName mutates receiver field 'name'.
func (c *Counter) SetName(n string) {
	c.name = n
}

// SetBoth mutates multiple receiver fields.
func (c *Counter) SetBoth(n string, v int) {
	c.count = v
	c.name = n
}

// Value is a value receiver — should NOT detect mutation.
func (c Counter) Value() int {
	return c.count
}

// ValueReceiverTrap writes to a value receiver copy — NOT a real
// mutation. Gaze should NOT report ReceiverMutation here.
func (c Counter) ValueReceiverTrap() {
	c.count++ // mutates the copy, not the original
}

// Normalize demonstrates pointer argument mutation.
func Normalize(v *[3]float64) {
	mag := v[0]*v[0] + v[1]*v[1] + v[2]*v[2]
	_ = mag
	v[0] = 1.0
}

// FillSlice mutates data through a pointer parameter.
func FillSlice(dst *[]int, val int) {
	*dst = append(*dst, val)
}

// ReadOnly takes a pointer but does NOT write through it.
func ReadOnly(v *int) int {
	return *v
}

// Config demonstrates nested struct for deep mutation.
type Config struct {
	Timeout int
	Nested  struct {
		Value string
	}
}

// UpdateConfig mutates a receiver field.
func (c *Config) UpdateConfig(timeout int) {
	c.Timeout = timeout
}

// UpdateNested mutates a nested field through the receiver.
func (c *Config) UpdateNested(v string) {
	c.Nested.Value = v
}
