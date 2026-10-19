package comissao

type OrigemPercentual string

const (
	OrigemPercentualPadrao OrigemPercentual = "padrao"
	OrigemPercentualManual OrigemPercentual = "manual"
)

func OrigemPercentualDe(valor string) OrigemPercentual {
	return OrigemPercentual(valor)
}

func (o OrigemPercentual) String() string {
	return string(o)
}
