package receber

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type CanalBoleto string

const (
	CanalBoletoEmail    CanalBoleto = "email"
	CanalBoletoWhatsapp CanalBoleto = "whatsapp"
	CanalBoletoPortal   CanalBoleto = "portal"
	CanalBoletoCorreio  CanalBoleto = "correio"
	CanalBoletoOutro    CanalBoleto = "outro"
)

func NovoCanalBoleto(valor string) (CanalBoleto, error) {
	canal := CanalBoleto(strings.ToLower(strings.TrimSpace(valor)))

	switch canal {
	case CanalBoletoEmail, CanalBoletoWhatsapp, CanalBoletoPortal, CanalBoletoCorreio, CanalBoletoOutro:
		return canal, nil
	case "":
		return "", domain.ErroValidacao("canal de envio do boleto é obrigatório")
	default:
		return "", domain.ErroValidacao("canal de envio do boleto deve ser email, whatsapp, portal, correio ou outro")
	}
}

func CanalBoletoDe(valor string) CanalBoleto {
	return CanalBoleto(valor)
}

func (c CanalBoleto) String() string {
	return string(c)
}
