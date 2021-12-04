package pgtype

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgio"
)

type Tsrange struct {
	Lower     Timestamp
	Upper     Timestamp
	LowerType BoundType
	UpperType BoundType
	Valid     bool
}

func (dst *Tsrange) Set(src interface{}) error {
	// untyped nil and typed nil interfaces are different
	if src == nil {
		*dst = Tsrange{}
		return nil
	}

	switch value := src.(type) {
	case Tsrange:
		*dst = value
	case *Tsrange:
		*dst = *value
	case string:
		return dst.DecodeText(nil, []byte(value))
	default:
		return fmt.Errorf("cannot convert %v to Tsrange", src)
	}

	return nil
}

func (src Tsrange) Get() interface{} {
	if !src.Valid {
		return nil
	}
	return src
}

func (src *Tsrange) AssignTo(dst interface{}) error {
	return fmt.Errorf("cannot assign %v to %T", src, dst)
}

func (dst *Tsrange) DecodeText(ci *ConnInfo, src []byte) error {
	if src == nil {
		*dst = Tsrange{}
		return nil
	}

	utr, err := ParseUntypedTextRange(string(src))
	if err != nil {
		return err
	}

	*dst = Tsrange{Valid: true}

	dst.LowerType = utr.LowerType
	dst.UpperType = utr.UpperType

	if dst.LowerType == Empty {
		return nil
	}

	if dst.LowerType == Inclusive || dst.LowerType == Exclusive {
		if err := dst.Lower.DecodeText(ci, []byte(utr.Lower)); err != nil {
			return err
		}
	}

	if dst.UpperType == Inclusive || dst.UpperType == Exclusive {
		if err := dst.Upper.DecodeText(ci, []byte(utr.Upper)); err != nil {
			return err
		}
	}

	return nil
}

func (dst *Tsrange) DecodeBinary(ci *ConnInfo, src []byte) error {
	if src == nil {
		*dst = Tsrange{}
		return nil
	}

	ubr, err := ParseUntypedBinaryRange(src)
	if err != nil {
		return err
	}

	*dst = Tsrange{Valid: true}

	dst.LowerType = ubr.LowerType
	dst.UpperType = ubr.UpperType

	if dst.LowerType == Empty {
		return nil
	}

	if dst.LowerType == Inclusive || dst.LowerType == Exclusive {
		if err := dst.Lower.DecodeBinary(ci, ubr.Lower); err != nil {
			return err
		}
	}

	if dst.UpperType == Inclusive || dst.UpperType == Exclusive {
		if err := dst.Upper.DecodeBinary(ci, ubr.Upper); err != nil {
			return err
		}
	}

	return nil
}

func (src Tsrange) EncodeText(ci *ConnInfo, buf []byte) ([]byte, error) {
	if !src.Valid {
		return nil, nil
	}

	switch src.LowerType {
	case Exclusive, Unbounded:
		buf = append(buf, '(')
	case Inclusive:
		buf = append(buf, '[')
	case Empty:
		return append(buf, "empty"...), nil
	default:
		return nil, fmt.Errorf("unknown lower bound type %v", src.LowerType)
	}

	var err error

	if src.LowerType != Unbounded {
		buf, err = src.Lower.EncodeText(ci, buf)
		if err != nil {
			return nil, err
		} else if buf == nil {
			return nil, fmt.Errorf("Lower cannot be null unless LowerType is Unbounded")
		}
	}

	buf = append(buf, ',')

	if src.UpperType != Unbounded {
		buf, err = src.Upper.EncodeText(ci, buf)
		if err != nil {
			return nil, err
		} else if buf == nil {
			return nil, fmt.Errorf("Upper cannot be null unless UpperType is Unbounded")
		}
	}

	switch src.UpperType {
	case Exclusive, Unbounded:
		buf = append(buf, ')')
	case Inclusive:
		buf = append(buf, ']')
	default:
		return nil, fmt.Errorf("unknown upper bound type %v", src.UpperType)
	}

	return buf, nil
}

func (src Tsrange) EncodeBinary(ci *ConnInfo, buf []byte) ([]byte, error) {
	if !src.Valid {
		return nil, nil
	}

	var rangeType byte
	switch src.LowerType {
	case Inclusive:
		rangeType |= lowerInclusiveMask
	case Unbounded:
		rangeType |= lowerUnboundedMask
	case Exclusive:
	case Empty:
		return append(buf, emptyMask), nil
	default:
		return nil, fmt.Errorf("unknown LowerType: %v", src.LowerType)
	}

	switch src.UpperType {
	case Inclusive:
		rangeType |= upperInclusiveMask
	case Unbounded:
		rangeType |= upperUnboundedMask
	case Exclusive:
	default:
		return nil, fmt.Errorf("unknown UpperType: %v", src.UpperType)
	}

	buf = append(buf, rangeType)

	var err error

	if src.LowerType != Unbounded {
		sp := len(buf)
		buf = pgio.AppendInt32(buf, -1)

		buf, err = src.Lower.EncodeBinary(ci, buf)
		if err != nil {
			return nil, err
		}
		if buf == nil {
			return nil, fmt.Errorf("Lower cannot be null unless LowerType is Unbounded")
		}

		pgio.SetInt32(buf[sp:], int32(len(buf[sp:])-4))
	}

	if src.UpperType != Unbounded {
		sp := len(buf)
		buf = pgio.AppendInt32(buf, -1)

		buf, err = src.Upper.EncodeBinary(ci, buf)
		if err != nil {
			return nil, err
		}
		if buf == nil {
			return nil, fmt.Errorf("Upper cannot be null unless UpperType is Unbounded")
		}

		pgio.SetInt32(buf[sp:], int32(len(buf[sp:])-4))
	}

	return buf, nil
}

// Scan implements the database/sql Scanner interface.
func (dst *Tsrange) Scan(src interface{}) error {
	if src == nil {
		*dst = Tsrange{}
		return nil
	}

	switch src := src.(type) {
	case string:
		return dst.DecodeText(nil, []byte(src))
	case []byte:
		srcCopy := make([]byte, len(src))
		copy(srcCopy, src)
		return dst.DecodeText(nil, srcCopy)
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (src Tsrange) Value() (driver.Value, error) {
	return EncodeValueText(src)
}
