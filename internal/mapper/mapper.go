package mapper

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
)

type DecoderOption func(d *Decoder)

func WithSeparator(separator string) DecoderOption {
	return func(d *Decoder) {
		d.separator = separator
	}
}

func WithTagDefaulter(f TagDefaulter) DecoderOption {
	return func(d *Decoder) {
		d.tagDefaulter = f
	}
}

func MapLookup(m map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		value, found := m[key]
		return value, found
	}
}

type TagDefaulter func(name string) string

type LookupFunc func(key string) (string, bool)

type Decoder struct {
	separator    string
	tagDefaulter TagDefaulter
	lookup       LookupFunc
}

func NewDecoder(lookup LookupFunc, options ...DecoderOption) *Decoder {
	dec := &Decoder{
		separator: "_",
		lookup:    lookup,
	}
	for _, opt := range options {
		opt(dec)
	}
	return dec
}

func (d *Decoder) Decode(v interface{}) error {
	_, err := d.decode("", reflect.ValueOf(v).Elem())
	return err
}

func (d *Decoder) decode(root string, v reflect.Value) (bool, error) {
	var found bool
	var err error
	switch v.Kind() {
	case reflect.Ptr:
		value := reflect.New(v.Elem().Type()).Elem()
		found, err = d.decode(root, value)
		if err != nil {
			return false, err
		}
		if found {
			v.Elem().Set(value)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			subKey, ok := v.Type().Field(i).Tag.Lookup("map")
			if !ok {
				// Try to get tag through the name
				subKey = v.Type().Field(i).Name
				if d.tagDefaulter != nil {
					subKey = d.tagDefaulter(subKey)
				}
			}
			if root != "" {
				subKey = root + d.separator + subKey
			}
			switch f.Kind() {
			case reflect.Struct:
				if found, err = d.decode(subKey, f); err != nil {
					return false, err
				}
			case reflect.Slice:
				if found, err = d.decode(subKey, f); err != nil {
					return false, err
				}
			case reflect.Ptr:
				value := reflect.New(f.Type().Elem())
				found, err := d.decode(subKey, value)
				if err != nil {
					return false, err
				}
				if found {
					f.Set(value)
				}
			default:
				if found, err = d.decodePrimitive(subKey, f); err != nil {
					return false, err
				}
			}
		}
	case reflect.Slice:
		sliceType := v.Type().Elem()
		found = true
		// Check if it might be a binary
		if sliceType.Kind() == reflect.Uint8 {
			// Attempt to decode as binary
			value, found := d.lookup(root)
			if found {
				bytes, err := base64.StdEncoding.DecodeString(value)
				if err != nil {
					return false, fmt.Errorf("failed decoding base64 string: %w", err)
				}
				v.SetBytes(bytes)
				return true, nil
			}
		}
		for i := 0; found; i++ {
			// Create new slice element
			f := reflect.New(sliceType).Elem()
			subKey := root + d.separator + strconv.Itoa(i)
			if found, err = d.decode(subKey, f); err != nil {
				return false, err
			} else if found {
				v.Set(reflect.Append(v, f))
			}
		}
		found = v.Len() > 0
	default:
		return d.decodePrimitive(root, v)
	}
	return found, nil
}

func (d *Decoder) decodePrimitive(key string, v reflect.Value) (bool, error) {
	value, found := d.lookup(key)
	if !found {
		return false, nil
	}
	switch v.Kind() {
	case reflect.Bool:
		decodedValue, err := strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("failed parsing bool: %w", err)
		}
		v.SetBool(decodedValue)
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		decodedValue, err := strconv.ParseInt(value, 10, v.Type().Bits())
		if err != nil {
			return false, fmt.Errorf("failed parsing int64: %w", err)
		}
		v.SetInt(decodedValue)
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		decodedValue, err := strconv.ParseUint(value, 10, v.Type().Bits())
		if err != nil {
			return false, fmt.Errorf("failed parsing uint64: %w", err)
		}
		v.SetUint(decodedValue)
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		decodedValue, err := strconv.ParseFloat(value, v.Type().Bits())
		if err != nil {
			return false, fmt.Errorf("failed parsing float64: %w", err)
		}
		v.SetFloat(decodedValue)
	case reflect.Complex64:
		fallthrough
	case reflect.Complex128:
		decodedValue, err := strconv.ParseComplex(value, v.Type().Bits())
		if err != nil {
			return false, fmt.Errorf("failed parsing complex128: %w", err)
		}
		v.SetComplex(decodedValue)
	case reflect.String:
		v.SetString(value)
	default:
		panic("Unknown type: " + v.Kind().String())
	}
	return true, nil
}
