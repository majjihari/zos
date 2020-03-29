package builder

import (
	"encoding/json"
	"io"

	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

// VolumeBuilder type
type VolumeBuilder struct {
	volume workloads.Volume
}

// NewVolumeBuilder creates a new volume builder
func NewVolumeBuilder(node string) *VolumeBuilder {
	return &VolumeBuilder{
		volume: workloads.Volume{
			NodeId: node,
			Size:   1,
			Type:   workloads.VolumeTypeHDD,
		},
	}
}

// LoadVolumeBuilder loads builder from input stream
func LoadVolumeBuilder(in io.Reader) (*VolumeBuilder, error) {
	var builder VolumeBuilder
	if err := json.NewDecoder(in).Decode(&builder.volume); err != nil {
		return nil, err
	}

	return &builder, nil
}

// WithSize sets size of the volume
func (b *VolumeBuilder) WithSize(size uint64) *VolumeBuilder {
	b.volume.Size = int64(size)
	return b
}

// WithType sets type of the volume
func (b *VolumeBuilder) WithType(typ workloads.VolumeTypeEnum) *VolumeBuilder {
	b.volume.Type = typ
	return b
}

// Save serializes builder information
func (b *VolumeBuilder) Save(out io.Writer) error {
	return json.NewEncoder(out).Encode(b.volume)
}

// Build gets the final Volume workload
func (b *VolumeBuilder) Build() workloads.Volume {
	return b.volume
}
