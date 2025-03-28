package models

import (
	"time"
)

type SectionStatus string

const (
	Scheduled SectionStatus = "Scheduled"
	Active    SectionStatus = "Active"
	Cancelled SectionStatus = "Cancelled"
	Completed SectionStatus = "Completed"
	Paused    SectionStatus = "Paused"
	Pending   SectionStatus = "Pending"
)

/*
ProgramSections are physical 'instances' of Programs,
with a collection of Events held at a particular Facility
*/
type ProgramSection struct {
	DatabaseFields
	ProgramID      uint          `json:"program_id" gorm:"not null"`
	FacilityID     uint          `json:"facility_id" gorm:"not null"`
	Capacity       int64         `json:"capacity" gorm:"not null"`
	Name           string        `json:"name" gorm:"size:255" validate:"required,max=255"`
	InstructorName string        `json:"instructor_name" gorm:"size:255" validate:"required,max=255"`
	Description    string        `json:"description" gorm:"not null" validate:"required,max=255"`
	ArchivedAt     *time.Time    `json:"archived_at"`
	StartDt        time.Time     `gorm:"type:date" json:"start_dt"`
	EndDt          *time.Time    `gorm:"type:date" json:"end_dt"`
	Status         SectionStatus `json:"section_status" gorm:"type:section_status" validate:"required"`
	CreditHours    *int64        `json:"credit_hours"`

	Program  *Program              `json:"program" gorm:"foreignKey:ProgramID;references:ID"`
	Facility *Facility             `json:"-" gorm:"foreignKey:FacilityID;references:ID"`
	Events   []ProgramSectionEvent `json:"events" gorm:"foreignKey:SectionID;references:ID"`
}

func (ProgramSection) TableName() string { return "program_sections" }

/*
ProgramSectionEnrollments is a User's enrollment in a particular Program's 'section' at their respective facility,
meaning they will need to attend the SectionEvents for that section: tracked by SectionEventAttendance
*/
type ProgramSectionEnrollment struct {
	DatabaseFields
	SectionID        uint   `json:"section_id" gorm:"not null"`
	UserID           uint   `json:"user_id" gorm:"not null"`
	EnrollmentStatus string `json:"enrollment_status" gorm:"size:255" validate:"max=255"`

	User    *User           `json:"user" gorm:"foreignKey:UserID;references:ID"`
	Section *ProgramSection `json:"section" gorm:"foreignKey:SectionID;references:ID"`
}

func (ProgramSectionEnrollment) TableName() string { return "program_section_enrollments" }

type ProgramSectionDetail struct {
	ID             int64      `json:"id"`
	FacilityName   string     `json:"facility_name"`
	InstructorName string     `json:"instructor_name"`
	StartDt        time.Time  `json:"start_dt"`
	Duration       string     `json:"duration"`
	Capacity       string     `json:"capacity"`
	Enrolled       int        `json:"enrolled"`
	EndDt          *time.Time `json:"end_dt"`
}
