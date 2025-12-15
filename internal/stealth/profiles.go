package stealth

type Profile struct {
	DelayMultiplier     float64
	JitterFactor        float64
	TypoRate            float64
	ImperfectionEnabled bool
	MaxDailyRisk        int
}

var ProfileCautious = Profile{
	DelayMultiplier:     1.5,
	JitterFactor:        0.3,
	TypoRate:            0.1,
	ImperfectionEnabled: true,
	MaxDailyRisk:        50,
}

var ProfileNormal = Profile{
	DelayMultiplier:     1.0,
	JitterFactor:        0.2,
	TypoRate:            0.3,
	ImperfectionEnabled: true,
	MaxDailyRisk:        100,
}

var ProfileAggressive = Profile{
	DelayMultiplier:     0.6,
	JitterFactor:        0.15,
	TypoRate:            0.5,
	ImperfectionEnabled: false,
	MaxDailyRisk:        200,
}
