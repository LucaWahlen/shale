package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"shale/internal/domain"
)

const ExportSchemaVersion = 3

type TransferService struct {
	schedules domain.ScheduleRepository
	events    domain.EventRepository
	attendees domain.AttendeeRepository
	settings  *SettingsService
	tx        domain.TransactionManager
	clock     Clock
}

func NewTransferService(
	schedules domain.ScheduleRepository,
	events domain.EventRepository,
	attendees domain.AttendeeRepository,
	settings *SettingsService,
	tx domain.TransactionManager,
	clock Clock,
) *TransferService {
	return &TransferService{
		schedules: schedules,
		events:    events,
		attendees: attendees,
		settings:  settings,
		tx:        tx,
		clock:     clock,
	}
}

type ExportDoc struct {
	App           string            `json:"app"`
	SchemaVersion int               `json:"schema_version"`
	ExportedAt    string            `json:"exported_at"`
	Settings      map[string]string `json:"settings"`
	Schedules     []ExportSchedule  `json:"schedules"`
}

type ExportSchedule struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Events      []ExportEvent `json:"events"`
}

type ExportEvent struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Location    string       `json:"location"`
	StartsAt    string       `json:"starts_at"`
	AllDay      bool         `json:"all_day"`
	EndsAt      string       `json:"ends_at,omitempty"`
	Attendees   []ExportName `json:"attendees"`
}

type ExportName struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ManageToken string `json:"manage_token"`
}

type ImportCounts struct {
	Schedules int64 `json:"schedules"`
	Events    int64 `json:"events"`
	Attendees int64 `json:"attendees"`
}

func (s *TransferService) Export(ctx context.Context) (ExportDoc, error) {
	values, err := s.settings.All(ctx)
	if err != nil {
		return ExportDoc{}, err
	}
	scheds, err := s.schedules.List(ctx)
	if err != nil {
		return ExportDoc{}, err
	}
	out := ExportDoc{
		App:           "shale",
		SchemaVersion: ExportSchemaVersion,
		ExportedAt:    s.clock().UTC().Format(time.RFC3339),
		Settings:      values,
		Schedules:     make([]ExportSchedule, 0, len(scheds)),
	}
	for _, sc := range scheds {
		events, err := s.events.ListBySchedule(ctx, sc.ID)
		if err != nil {
			return ExportDoc{}, err
		}
		es := ExportSchedule{
			ID:          sc.ID,
			Title:       sc.Title,
			Description: sc.Description,
			Events:      make([]ExportEvent, 0, len(events)),
		}
		for _, ev := range events {
			atts, err := s.attendees.ListByEvent(ctx, ev.ID)
			if err != nil {
				return ExportDoc{}, err
			}
			ee := ExportEvent{
				ID:          ev.ID,
				Name:        ev.Name,
				Description: ev.Description,
				Location:    ev.Location,
				StartsAt:    ev.StartsAt,
				AllDay:      ev.AllDay,
				EndsAt:      ev.EndsAt,
				Attendees:   make([]ExportName, 0, len(atts)),
			}
			for _, a := range atts {
				ee.Attendees = append(ee.Attendees, ExportName{ID: a.ID, Name: a.Name, ManageToken: a.ManageToken})
			}
			es.Events = append(es.Events, ee)
		}
		out.Schedules = append(out.Schedules, es)
	}
	return out, nil
}

type importDoc struct {
	App           string            `json:"app"`
	SchemaVersion *int              `json:"schema_version"`
	Settings      map[string]string `json:"settings"`
	Schedules     []importSchedule  `json:"schedules"`
}

type importSchedule struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Events      []importEvent `json:"events"`
}

type importEvent struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Location    string       `json:"location"`
	StartsAt    string       `json:"starts_at"`
	AllDay      bool         `json:"all_day"`
	EndsAt      string       `json:"ends_at"`
	Attendees   []ExportName `json:"attendees"`
}

func (s *TransferService) Import(ctx context.Context, raw []byte) (ImportCounts, error) {
	doc, details, err := parseImportDoc(raw)
	if err != nil {
		return ImportCounts{}, err
	}
	if len(details) > 0 {
		return ImportCounts{}, domain.NewErrorWithDetails(domain.KindUnprocessable, "import validation failed", details)
	}
	counts := ImportCounts{}
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.schedules.DeleteAll(ctx); err != nil {
			return err
		}
		if err := s.settings.ReplaceAll(ctx, doc.Settings); err != nil {
			return err
		}
		for _, is := range doc.Schedules {
			sched := domain.Schedule{
				ID:          is.ID,
				Title:       strings.TrimSpace(is.Title),
				Description: is.Description,
			}
			if err := s.schedules.Create(ctx, &sched); err != nil {
				return err
			}
			counts.Schedules++
			for _, ie := range is.Events {
				ev := domain.Event{
					ID:          ie.ID,
					ScheduleID:  sched.ID,
					Name:        ie.Name,
					Description: ie.Description,
					Location:    ie.Location,
					StartsAt:    ie.StartsAt,
					AllDay:      ie.AllDay,
					EndsAt:      ie.EndsAt,
				}
				if err := s.events.Create(ctx, &ev); err != nil {
					return err
				}
				counts.Events++
				seen := map[string]bool{}
				for _, ia := range ie.Attendees {
					name, verr := domain.ValidateName(ia.Name)
					if verr != nil {
						return verr
					}
					normalized := domain.NormalizeName(name)
					if seen[normalized] {
						continue
					}
					seen[normalized] = true
					token := ia.ManageToken
					if token == "" {
						var terr error
						token, terr = domain.NewManageToken()
						if terr != nil {
							return terr
						}
					}
					att := domain.Attendee{
						ID:             ia.ID,
						EventID:        ev.ID,
						Name:           name,
						NormalizedName: normalized,
						ManageToken:    token,
					}
					if err := s.attendees.Create(ctx, &att); err != nil {
						return err
					}
					counts.Attendees++
				}
			}
		}
		return nil
	})
	if err != nil {
		return ImportCounts{}, err
	}
	return counts, nil
}

func parseImportDoc(raw []byte) (importDoc, []domain.Detail, error) {
	var doc importDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return importDoc{}, nil, domain.NewError(domain.KindInvalid, "body must be a valid JSON export document")
	}
	if doc.App != "shale" {
		return importDoc{}, nil, domain.NewError(domain.KindInvalid, "not a shale export: missing app=shale marker")
	}
	var details []domain.Detail
	if doc.SchemaVersion == nil {
		details = append(details, domain.Detail{Path: "schema_version", Message: "schema_version is required"})
	} else if *doc.SchemaVersion > ExportSchemaVersion {
		details = append(details, domain.Detail{
			Path:    "schema_version",
			Message: fmt.Sprintf("unsupported schema_version %d (supported: <= %d)", *doc.SchemaVersion, ExportSchemaVersion),
		})
	} else if *doc.SchemaVersion < 1 {
		details = append(details, domain.Detail{Path: "schema_version", Message: "schema_version must be >= 1"})
	}

	if doc.Settings == nil {
		doc.Settings = map[string]string{
			domain.SettingDefaultLanguage: domain.LanguageEN,
			domain.SettingAppName:         domain.DefaultAppName,
		}
	}
	if lang, ok := doc.Settings[domain.SettingDefaultLanguage]; ok && !domain.ValidLanguage(lang) {
		details = append(details, domain.Detail{Path: "settings.default_language", Message: "default_language must be 'en' or 'de'"})
	}
	if name, ok := doc.Settings[domain.SettingAppName]; ok {
		trimmed := strings.TrimSpace(name)
		switch {
		case trimmed == "":
			details = append(details, domain.Detail{Path: "settings.app_name", Message: "app_name must not be empty"})
		case runeLen(trimmed) > MaxAppNameRunes:
			details = append(details, domain.Detail{Path: "settings.app_name", Message: fmt.Sprintf("app_name must be at most %d characters", MaxAppNameRunes)})
		default:
			doc.Settings[domain.SettingAppName] = trimmed
		}
	}

	seenScheduleIDs := map[string]bool{}
	seenEventIDs := map[string]bool{}
	seenAttendeeIDs := map[string]bool{}
	seenTokens := map[string]bool{}

	for i, is := range doc.Schedules {
		base := fmt.Sprintf("schedules[%d]", i)
		if is.ID != "" {
			if !domain.ValidID(is.ID) {
				details = append(details, domain.Detail{Path: base + ".id", Message: "id must be a valid UUID"})
			} else if seenScheduleIDs[is.ID] {
				details = append(details, domain.Detail{Path: base + ".id", Message: "duplicate schedule id"})
			}
			seenScheduleIDs[is.ID] = true
		}
		if strings.TrimSpace(is.Title) == "" {
			details = append(details, domain.Detail{Path: base + ".title", Message: "title must not be empty"})
		}
		if runeLen(is.Title) > maxTitleRunes {
			details = append(details, domain.Detail{Path: base + ".title", Message: fmt.Sprintf("title must be at most %d characters", maxTitleRunes)})
		}
		if runeLen(is.Description) > maxDescriptionRunes {
			details = append(details, domain.Detail{Path: base + ".description", Message: "description too long"})
		}
		for j, ie := range is.Events {
			ebase := fmt.Sprintf("%s.events[%d]", base, j)
			if ie.ID != "" {
				if !domain.ValidID(ie.ID) {
					details = append(details, domain.Detail{Path: ebase + ".id", Message: "id must be a valid UUID"})
				} else if seenEventIDs[ie.ID] {
					details = append(details, domain.Detail{Path: ebase + ".id", Message: "duplicate event id"})
				}
				seenEventIDs[ie.ID] = true
			}
			if strings.TrimSpace(ie.Name) == "" {
				details = append(details, domain.Detail{Path: ebase + ".name", Message: "name must not be empty"})
			}
			if !domain.ValidStartsAt(ie.StartsAt) {
				details = append(details, domain.Detail{Path: ebase + ".starts_at", Message: "starts_at must be in YYYY-MM-DDTHH:MM form"})
			}
			if ie.AllDay {
				if ie.EndsAt != "" {
					details = append(details, domain.Detail{Path: ebase + ".ends_at", Message: "all-day events must not have an end time"})
				}
			} else if ie.EndsAt != "" && !domain.ValidEndsAt(ie.StartsAt, ie.EndsAt) {
				details = append(details, domain.Detail{Path: ebase + ".ends_at", Message: "ends_at must be in YYYY-MM-DDTHH:MM form and after starts_at"})
			}
			seenNames := map[string]bool{}
			for k, ia := range ie.Attendees {
				abase := fmt.Sprintf("%s.attendees[%d]", ebase, k)
				trimmed := strings.TrimSpace(ia.Name)
				if trimmed == "" {
					details = append(details, domain.Detail{Path: abase + ".name", Message: "name must not be empty"})
					continue
				}
				if runeLen(trimmed) > domain.MaxNameRunes {
					details = append(details, domain.Detail{Path: abase + ".name", Message: fmt.Sprintf("name must be at most %d characters", domain.MaxNameRunes)})
					continue
				}
				if ia.ID != "" {
					if !domain.ValidID(ia.ID) {
						details = append(details, domain.Detail{Path: abase + ".id", Message: "id must be a valid UUID"})
					} else if seenAttendeeIDs[ia.ID] {
						details = append(details, domain.Detail{Path: abase + ".id", Message: "duplicate attendee id"})
					}
					seenAttendeeIDs[ia.ID] = true
				}
				if ia.ManageToken != "" {
					if !domain.ValidManageToken(ia.ManageToken) {
						details = append(details, domain.Detail{Path: abase + ".manage_token", Message: "manage_token must be 32 hex characters"})
					} else if seenTokens[ia.ManageToken] {
						details = append(details, domain.Detail{Path: abase + ".manage_token", Message: "duplicate manage_token"})
					}
					seenTokens[ia.ManageToken] = true
				}
				norm := domain.NormalizeName(trimmed)
				if seenNames[norm] {
					details = append(details, domain.Detail{Path: abase + ".name", Message: "duplicate attendee name in event"})
				}
				seenNames[norm] = true
			}
		}
	}

	return doc, details, nil
}
