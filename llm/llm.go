// Package llm defines interfaces implemented by LLMs (or LLM-related services).
package llm

import (
	"context"
	"encoding/binary"
	"math"
)

type Schema struct {
	// Required. Data type.
	Type Type
	// Optional. The format of the data. This is used only for primitive
	// datatypes. Supported formats:
	//
	//	for NUMBER type: float, double
	//	for INTEGER type: int32, int64
	Format string
	// Optional. A brief description of the parameter. This could contain examples
	// of use. Parameter description may be formatted as Markdown.
	Description string
	// Optional. Indicates if the value may be null.
	Nullable bool
	// Optional. Possible values of the element of Type.STRING with enum format.
	// For example we can define an Enum Direction as :
	// {type:STRING, format:enum, enum:["EAST", NORTH", "SOUTH", "WEST"]}
	Enum []string
	// Optional. Schema of the elements of Type.ARRAY.
	Items *Schema
	// Optional. Properties of Type.OBJECT.
	Properties map[string]*Schema
	// Optional. Required properties of Type.OBJECT.
	Required []string
}
type Type int32

const (
	// TypeUnspecified means not specified, should not be used.
	TypeUnspecified Type = 0
	// TypeString means string type.
	TypeString Type = 1
	// TypeNumber means number type.
	TypeNumber Type = 2
	// TypeInteger means integer type.
	TypeInteger Type = 3
	// TypeBoolean means boolean type.
	TypeBoolean Type = 4
	// TypeArray means array type.
	TypeArray Type = 5
	// TypeObject means object type.
	TypeObject Type = 6
)

// An Embedder computes vector embeddings of a list of documents.
//
// EmbedDocs accepts an arbitrary number of documents and returns
// their embeddings. If the underlying implementation has a limit on
// the batch size, it should make multiple requests in order to process
// all the documents. If an error occurs after some, but not all, documents
// have been processed, EmbedDocs can return an error along with a
// shortened vector slice giving the vectors for a prefix of the document slice.
//
// See [QuoteEmbedder] for a semantically useless embedder that
// can nonetheless be helpful when writing tests,
// and see [github.com/superryanguo/ryai/gcp/gemini] for a real implementation.
type Embedder interface {
	EmbedDocs(ctx context.Context, docs []EmbedDoc) ([]Vector, error)
}

// An EmbedDoc is a single document to be embedded.
type EmbedDoc struct {
	Title string // title of document (optional)
	Text  string // text of document
}

// A Vector is an embedding vector, typically a high-dimensional unit vector.
type Vector []float32

// Dot returns the dot product of v and w.
//
// TODO(rsc): Using a float64 for the result is slightly higher
// precision and may be worth doing in the intermediate calculation
// but may not be worth the type conversions involved to return a float64.
// Perhaps the return type should still be float32 even if the math is float64.
func (v Vector) Dot(w Vector) float64 {
	v = v[:min(len(v), len(w))]
	w = w[:len(v)] // make "i in range for v" imply "i in range for w" to remove bounds check in loop
	t := float64(0)
	for i := range v {
		t += float64(v[i]) * float64(w[i])
	}
	return t
}

// Encode returns a byte encoding of the vector v,
// suitable for storing in a database.
func (v Vector) Encode() []byte {
	val := make([]byte, 4*len(v))
	for i, f := range v {
		binary.BigEndian.PutUint32(val[4*i:], math.Float32bits(f))
	}
	return val
}

// Decode decodes the byte encoding enc into the vector v.
// Enc should be a multiple of 4 bytes; any trailing bytes are ignored.
func (v *Vector) Decode(enc []byte) {
	if len(*v) < len(enc)/4 {
		*v = make(Vector, len(enc)/4)
	}
	*v = (*v)[:0]
	for ; len(enc) >= 4; enc = enc[4:] {
		*v = append(*v, math.Float32frombits(binary.BigEndian.Uint32(enc)))
	}
}

// A Part is part of a prompt to a [ContentGenerator].
type Part interface {
	isPart()
}

// A Text is a [Part] that is a string.
type Text string

// A Blob is a [Part] that is binary data, like an image or video.
type Blob struct {
	MIMEType string
	Data     []byte
}

func (Text) isPart() {}
func (Blob) isPart() {}

// A ContentGenerator generates content in response to prompts.
//
// See [EchoContentGenerator] for a generator, useful for testing, that
// always responds with a deterministic message derived from the prompts.
//
// See [github.com/superryanguo/ryai/gcp/gemini] for a real implementation.
type ContentGenerator interface {
	// Model returns the name of the generative model
	// used by this ContentGenerator.
	Model() string
	// GenerateContent generates a text response given a JSON schema
	// and one or more prompt parts.
	// If the JSON schema is nil, GenerateContent outputs a plain text response.
	GenerateContent(ctx context.Context, schema *Schema, parts []Part) (string, error)
	// SetTemperature changes the temperature of the model.
	SetTemperature(float32)
}
