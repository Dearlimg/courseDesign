package ga

import (
	"bytes"
	"encoding/json"
)

// UnmarshalJSON distinguishes an explicit zero from an omitted parameter.
func (p *Params) UnmarshalJSON(data []byte) error {
	type plain Params
	var decoded plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*p = Params(decoded)
	_, p.crossSet = fields["crossoverRate"]
	_, p.mutationSet = fields["mutationRate"]
	_, p.elitismSet = fields["elitism"]
	_, p.seedSet = fields["seed"]
	return nil
}
