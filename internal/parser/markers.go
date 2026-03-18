package parser

import "regexp"

var (
	reProper  = regexp.MustCompile(`(?i)\bproper\d?\b`)
	reProper2 = regexp.MustCompile(`(?i)\bproper2\b`)
	reRepack  = regexp.MustCompile(`(?i)\brepack\d?\b`)
	reRepack2 = regexp.MustCompile(`(?i)\brepack2\b`)
	reRerip   = regexp.MustCompile(`(?i)\brerip\d?\b`)
	reRerip2  = regexp.MustCompile(`(?i)\brerip2\b`)
	reReal    = regexp.MustCompile(`(?i)\breal\b`)

	reHybrid       = regexp.MustCompile(`(?i)\bhybrid\b`)
	re3D           = regexp.MustCompile(`(?i)(?:\b3d\b|\bsbs\b|\bhsbs\b|\bhou\b|\bhalf[\s._-]?ou\b)`)
	reHardcodedSub = regexp.MustCompile(`(?i)(?:\bhc\b|\bhardcoded\b|\bhardsub\b|\bkorsub\b)`)
	reSample       = regexp.MustCompile(`(?i)\bsample\b`)
	reInternal     = regexp.MustCompile(`(?i)\binternal\b`)
	reLimited      = regexp.MustCompile(`(?i)\blimited\b`)
	reSubbed       = regexp.MustCompile(`(?i)\bsubbed\b`)
	reDubbed       = regexp.MustCompile(`(?i)\bdubbed\b`)
)

func parseRevision(norm string) Revision {
	rev := Revision{Version: 1}
	switch {
	case reProper2.MatchString(norm) || reRepack2.MatchString(norm) || reRerip2.MatchString(norm):
		rev.Version = 3
	case reProper.MatchString(norm) || reRepack.MatchString(norm) || reRerip.MatchString(norm):
		rev.Version = 2
	}
	if reReal.MatchString(norm) {
		rev.IsReal = true
	}
	return rev
}

func parseMarkers(norm string, p *ParsedRelease) {
	p.IsHybrid = reHybrid.MatchString(norm)
	p.Is3D = re3D.MatchString(norm)
	p.IsHardcodedSub = reHardcodedSub.MatchString(norm)
	p.IsSample = reSample.MatchString(norm)
	p.IsInternal = reInternal.MatchString(norm)
	p.IsLimited = reLimited.MatchString(norm)
	p.IsSubbed = reSubbed.MatchString(norm)
	p.IsDubbed = reDubbed.MatchString(norm)
	p.IsProper = reProper.MatchString(norm)
	p.IsRepack = reRepack.MatchString(norm) || reRerip.MatchString(norm)
}
