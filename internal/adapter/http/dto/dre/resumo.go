package dre

import (
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

type ItemResponse struct {
	Chave         string     `json:"chave"`
	CategoriaID   *uuid.UUID `json:"categoria_id"`
	Nome          string     `json:"nome"`
	ValorCentavos int64      `json:"valor_centavos"`
	Quantidade    int        `json:"quantidade"`
}

type LinhaResponse struct {
	Linha         string         `json:"linha"`
	ValorCentavos int64          `json:"valor_centavos"`
	Quantidade    int            `json:"quantidade"`
	Itens         []ItemResponse `json:"itens"`
}

type SubtotaisResponse struct {
	ReceitaLiquidaCentavos       int64 `json:"receita_liquida_centavos"`
	LucroBrutoCentavos           int64 `json:"lucro_bruto_centavos"`
	MargemContribuicaoCentavos   int64 `json:"margem_contribuicao_centavos"`
	ResultadoOperacionalCentavos int64 `json:"resultado_operacional_centavos"`
	ResultadoCentavos            int64 `json:"resultado_centavos"`
}

type IndicadoresResponse struct {
	MargemContribuicaoBP    *int64 `json:"margem_contribuicao_bp"`
	PontoEquilibrioCentavos *int64 `json:"ponto_equilibrio_centavos"`
}

type DemonstrativoResponse struct {
	Linhas             []LinhaResponse     `json:"linhas"`
	Subtotais          SubtotaisResponse   `json:"subtotais"`
	Indicadores        IndicadoresResponse `json:"indicadores"`
	QuantidadeSemCusto int64               `json:"quantidade_sem_custo"`
}

type DemonstrativoEmpresaResponse struct {
	EmpresaID uuid.UUID `json:"empresa_id"`
	DemonstrativoResponse
}

type MesDREResponse struct {
	Mes         string                         `json:"mes"`
	Empresas    []DemonstrativoEmpresaResponse `json:"empresas"`
	Consolidado DemonstrativoResponse          `json:"consolidado"`
}

type TotalDREResponse struct {
	Empresas    []DemonstrativoEmpresaResponse `json:"empresas"`
	Consolidado DemonstrativoResponse          `json:"consolidado"`
}

type ResumoDREResponse struct {
	De    string           `json:"de"`
	Ate   string           `json:"ate"`
	Meses []MesDREResponse `json:"meses"`
	Total TotalDREResponse `json:"total"`
}

func NovoResumoDREResponse(resumo domaindre.ResumoDRE) ResumoDREResponse {
	resposta := ResumoDREResponse{
		De:    resumo.De.String(),
		Ate:   resumo.Ate.String(),
		Meses: make([]MesDREResponse, 0, len(resumo.Meses)),
		Total: TotalDREResponse{
			Empresas:    novasEmpresas(resumo.Total.Empresas),
			Consolidado: novoDemonstrativo(resumo.Total.Consolidado),
		},
	}

	for _, mes := range resumo.Meses {
		resposta.Meses = append(resposta.Meses, MesDREResponse{
			Mes:         mes.Mes.String(),
			Empresas:    novasEmpresas(mes.Empresas),
			Consolidado: novoDemonstrativo(mes.Consolidado),
		})
	}

	return resposta
}

func novasEmpresas(empresas []domaindre.DemonstrativoEmpresa) []DemonstrativoEmpresaResponse {
	respostas := make([]DemonstrativoEmpresaResponse, 0, len(empresas))
	for _, empresa := range empresas {
		respostas = append(respostas, DemonstrativoEmpresaResponse{
			EmpresaID:             empresa.EmpresaID,
			DemonstrativoResponse: novoDemonstrativo(empresa.Demonstrativo),
		})
	}

	return respostas
}

func novoDemonstrativo(demonstrativo domaindre.Demonstrativo) DemonstrativoResponse {
	resposta := DemonstrativoResponse{
		Linhas: make([]LinhaResponse, 0, len(demonstrativo.Linhas)),
		Subtotais: SubtotaisResponse{
			ReceitaLiquidaCentavos:       demonstrativo.Subtotais.ReceitaLiquidaCentavos,
			LucroBrutoCentavos:           demonstrativo.Subtotais.LucroBrutoCentavos,
			MargemContribuicaoCentavos:   demonstrativo.Subtotais.MargemContribuicaoCentavos,
			ResultadoOperacionalCentavos: demonstrativo.Subtotais.ResultadoOperacionalCentavos,
			ResultadoCentavos:            demonstrativo.Subtotais.ResultadoCentavos,
		},
		Indicadores: IndicadoresResponse{
			MargemContribuicaoBP:    demonstrativo.Indicadores.MargemContribuicaoBP,
			PontoEquilibrioCentavos: demonstrativo.Indicadores.PontoEquilibrioCentavos,
		},
		QuantidadeSemCusto: demonstrativo.QuantidadeSemCusto,
	}

	for _, linha := range demonstrativo.Linhas {
		itens := make([]ItemResponse, 0, len(linha.Itens))
		for _, item := range linha.Itens {
			itens = append(itens, ItemResponse{
				Chave:         item.Chave,
				CategoriaID:   item.CategoriaID,
				Nome:          item.Nome,
				ValorCentavos: item.ValorCentavos,
				Quantidade:    item.Quantidade,
			})
		}
		resposta.Linhas = append(resposta.Linhas, LinhaResponse{
			Linha:         linha.Linha.String(),
			ValorCentavos: linha.ValorCentavos,
			Quantidade:    linha.Quantidade,
			Itens:         itens,
		})
	}

	return resposta
}
