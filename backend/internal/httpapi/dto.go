package httpapi

type publicAttendeeDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type publicEventDTO struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Location    string              `json:"location"`
	StartsAt    string              `json:"starts_at"`
	AllDay      bool                `json:"all_day"`
	EndsAt      string              `json:"ends_at"`
	Attendable  bool                `json:"attendable"`
	Attendees   []publicAttendeeDTO `json:"attendees"`
}

type publicScheduleDTO struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Events      []publicEventDTO `json:"events"`
}

type attendResponseDTO struct {
	Attendee    publicAttendeeDTO `json:"attendee"`
	ManageToken string            `json:"manage_token"`
	Created     bool              `json:"created"`
}

type adminAttendeeDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type adminEventDTO struct {
	ID          string             `json:"id"`
	ScheduleID  string             `json:"schedule_id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Location    string             `json:"location"`
	StartsAt    string             `json:"starts_at"`
	AllDay      bool               `json:"all_day"`
	EndsAt      string             `json:"ends_at"`
	Attendees   []adminAttendeeDTO `json:"attendees"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

type adminScheduleDTO struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	EventCount    int             `json:"event_count"`
	FirstStartsAt string          `json:"first_starts_at,omitempty"`
	LastStartsAt  string          `json:"last_starts_at,omitempty"`
	IsPast        bool            `json:"is_past"`
	Events        []adminEventDTO `json:"events"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

type adminSchedulePageDTO struct {
	Items    []adminScheduleDTO `json:"items"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type importResponseDTO struct {
	Imported importCountsDTO `json:"imported"`
}

type importCountsDTO struct {
	Schedules int64 `json:"schedules"`
	Events    int64 `json:"events"`
	Attendees int64 `json:"attendees"`
}

type auditEntryDTO struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	Action        string `json:"action"`
	Actor         string `json:"actor"`
	ActorName     string `json:"actor_name"`
	ScheduleID    string `json:"schedule_id"`
	ScheduleTitle string `json:"schedule_title"`
	EventID       string `json:"event_id"`
	EventName     string `json:"event_name"`
	Detail        string `json:"detail"`
}

type auditPageDTO struct {
	Items    []auditEntryDTO `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}
