package bench

import "time"

type Mode string

const (
	Fast Mode = "fast"
	Full Mode = "full"
)

type Direction string

const (
	Encrypt Direction = "encrypt"
	Decrypt Direction = "decrypt"
)

type CacheMode string

const (
	Hot  CacheMode = "hot"
	Cold CacheMode = "cold"
)

type Sample struct {
	Library    string    `json:"library"`
	Direction  Direction `json:"direction"`
	Cache      CacheMode `json:"cache"`
	Size       int       `json:"size"`
	Round      int       `json:"round"`
	Position   int       `json:"position"`
	Order      string    `json:"order"`
	ElapsedNS  int64     `json:"elapsed_ns"`
	Iterations int       `json:"iterations"`
	Checksum   uint64    `json:"checksum"`
	At         time.Time `json:"at"`
}

type Qualification struct {
	Library   string `json:"library"`
	Available bool   `json:"available"`
	Qualified bool   `json:"qualified"`
	Version   string `json:"version,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type PairResult struct {
	Library       string    `json:"library"`
	Direction     Direction `json:"direction"`
	Cache         CacheMode `json:"cache"`
	Size          int       `json:"size"`
	Ratios        []float64 `json:"ratios"`
	GeometricMean float64   `json:"geometric_mean"`
	CILow         float64   `json:"ci_low"`
	CIHigh        float64   `json:"ci_high"`
	Verdict       string    `json:"verdict"`
	Noise         string    `json:"noise,omitempty"`
	Drift         string    `json:"drift,omitempty"`
}

type Environment struct {
	OS             string   `json:"os"`
	Arch           string   `json:"arch"`
	GoVersion      string   `json:"go_version"`
	CPUs           int      `json:"cpus"`
	CPUModel       string   `json:"cpu_model,omitempty"`
	Acceleration   string   `json:"acceleration"`
	Virtualization string   `json:"virtualization"`
	Power          string   `json:"power"`
	Load           string   `json:"load"`
	Warnings       []string `json:"warnings"`
}

type Report struct {
	Schema         string          `json:"schema"`
	Started        time.Time       `json:"started"`
	Finished       time.Time       `json:"finished"`
	Mode           Mode            `json:"mode"`
	Environment    Environment     `json:"environment"`
	Qualifications []Qualification `json:"qualifications"`
	Samples        []Sample        `json:"samples"`
	Pairs          []PairResult    `json:"pairs"`
	Warnings       []string        `json:"warnings"`
	Methodology    string          `json:"methodology"`
}

type Options struct {
	Mode     Mode
	NoPython bool
	Progress func(string)
}
