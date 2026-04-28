// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package variable

import (
	"strconv"
	"time"
)

// SystemNamespace exposes UTC clock keys: current_date, current_time,
// current_datetime, day_of_week, date_rfc1123, date_unix, date_unix_ms.
type SystemNamespace struct{}

func (n *SystemNamespace) Get(suffix string, src Source, _ ResolveContext) (any, bool) {
	v, ok := n.fields(src.Now())[suffix]
	return v, ok
}

func (n *SystemNamespace) Enumerate(src Source, _ ResolveContext) map[string]any {
	return n.fields(src.Now())
}

func (n *SystemNamespace) fields(now time.Time) map[string]any {
	now = now.UTC()
	return map[string]any{
		"current_date":     now.Format("2006-01-02"),
		"current_time":     now.Format("15:04:05"),
		"current_datetime": now.Format(time.RFC3339),
		"day_of_week":      now.Weekday().String(),
		"date_rfc1123":     now.Format(time.RFC1123),
		"date_unix":        strconv.FormatInt(now.Unix(), 10),
		"date_unix_ms":     strconv.FormatInt(now.UnixMilli(), 10),
	}
}
