package service

import "time"

type Clock func() time.Time

func systemClock() time.Time { return time.Now() }
