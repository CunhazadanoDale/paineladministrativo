package pedido

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type TipoFrete string

const (
	TipoFreteCIF TipoFrete = "cif"
	TipoFreteFOB TipoFrete = "fob"
)

func NovoTipoFrete(valor string) (TipoFrete, error) {
	tipo := TipoFrete(strings.ToLower(strings.TrimSpace(valor)))

	if err := tipo.Validado(); err != nil {
		return "", err
	}

	return tipo, nil
}

func TipoFreteDe(valor string) TipoFrete {
	return TipoFrete(valor)
}

func (t TipoFrete) Validado() error {
	switch t {
	case TipoFreteCIF, TipoFreteFOB:
		return nil
	default:
		return domain.ErroValidacao("tipo de frete deve ser cif ou fob")
	}
}

func (t TipoFrete) String() string {
	return string(t)
}

type Frete struct {
	tipo     TipoFrete
	centavos *int64
}

func NovoFrete(tipo TipoFrete, centavos *int64) (Frete, error) {
	if err := tipo.Validado(); err != nil {
		return Frete{}, err
	}
	if centavos != nil && *centavos < 0 {
		return Frete{}, domain.ErroValidacao("valor do frete não pode ser negativo")
	}
	if tipo == TipoFreteCIF && centavos == nil {
		return Frete{}, domain.ErroValidacao("valor do frete é obrigatório no frete CIF")
	}

	return FreteDe(tipo, centavos), nil
}

func FreteDe(tipo TipoFrete, centavos *int64) Frete {
	frete := Frete{tipo: tipo}
	if centavos != nil {
		valor := *centavos
		frete.centavos = &valor
	}

	return frete
}

func (f Frete) Tipo() TipoFrete {
	return f.tipo
}

func (f Frete) Centavos() *int64 {
	if f.centavos == nil {
		return nil
	}

	valor := *f.centavos

	return &valor
}
