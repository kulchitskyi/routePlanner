package models

import (
	"fmt"
	"errors"
	"time"
	"math"
	"encoding/hex"
	"encoding/binary"
	"database/sql/driver"
	
	"gorm.io/gorm"
)

type Location struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
}

type PlaceCreateRequest struct {
	Name        string   `json:"name" validate:"required,min=5,max=100,safe_content"`
	Tags        []string `json:"tags"`
	Description string   `json:"description" validate:"required,min=10,max=1000,safe_content"`
	Location    Location `json:"location" validate:"required"`
	Address     string   `json:"address" validate:"required,max=255"`
	Rating      float64  `json:"rating" validate:"required,min=1,max=5"`
}

type Place struct {
	ID          string   `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name        string   `gorm:"type:varchar(255);not null" json:"name"`
	Tags        []string `gorm:"serializer:json;type:jsonb;index:,type:gin" json:"tags"`
	Description string   `gorm:"type:text" json:"description"`
	Location    Location `gorm:"type:geography(Point,4326)" json:"location"`
	Address     string   `gorm:"type:varchar(255)" json:"address"`
	Rating      float64  `gorm:"type:numeric" json:"rating"`
	CreatedBy   string   `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000000';index" json:"created_by"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type PlaceUpdateRequest struct {
	Name        *string   `json:"name,omitempty" validate:"omitempty,min=5,max=100,safe_content"`
	Description *string   `json:"description,omitempty" validate:"omitempty,min=10,max=1000,safe_content"`
	Location    *Location `json:"location,omitempty" validate:"omitempty"`
	Address     *string   `json:"address,omitempty" validate:"omitempty,max=255"`
	Rating      *float64  `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
}

func (l Location) Value() (driver.Value, error) {
	return fmt.Sprintf("SRID=4326;POINT(%f %f)", l.Longitude, l.Latitude), nil
}

func (l *Location) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bin []byte
	switch v := value.(type) {
	case []byte:
		bin = v
	case string:
		var err error
		bin, err = hex.DecodeString(v)
		if err != nil {
			return err
		}
	default:
		return errors.New("invalid type for Location Scan")
	}

	if len(bin) < 21 {
		return errors.New("invalid EWKB length for Point")
	}

	var order binary.ByteOrder
	if bin[0] == 1 {
		order = binary.LittleEndian
	} else {
		order = binary.BigEndian
	}

	offset := len(bin) - 16

	lonBits := order.Uint64(bin[offset+8 : offset+16])
	l.Latitude = math.Float64frombits(lonBits)

	latBits := order.Uint64(bin[offset : offset+8])
	l.Longitude = math.Float64frombits(latBits)

	return nil
}
