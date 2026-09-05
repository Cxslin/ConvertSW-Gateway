package runtime

import (
	"encoding/json"
	"io"
)

type StreamPresenter struct {
	w       io.Writer
	metrics StreamMetrics
}

func NewStreamPresenter(w io.Writer) *StreamPresenter {
	return &StreamPresenter{w: w}
}

func (p *StreamPresenter) Send(payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = p.w.Write([]byte("data: " + string(raw) + "\n\n"))
	return err
}

func (p *StreamPresenter) Done() error {
	_, err := p.w.Write([]byte("data: [DONE]\n\n"))
	return err
}

func (p *StreamPresenter) Metrics() StreamMetrics {
	return p.metrics
}
