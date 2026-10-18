package receber

import (
	"fmt"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	taxaMaximaPontosBase              = 10000
	tamanhoMaximoNumeroBordero        = 30
	tamanhoMaximoMotivoEstornoBordero = 500
)

type StatusAntecipacao string

const (
	StatusAntecipacaoAtiva     StatusAntecipacao = "ativa"
	StatusAntecipacaoEstornada StatusAntecipacao = "estornada"
)

func NovoStatusAntecipacao(valor string) (StatusAntecipacao, error) {
	status := StatusAntecipacao(strings.ToLower(strings.TrimSpace(valor)))

	switch status {
	case StatusAntecipacaoAtiva, StatusAntecipacaoEstornada:
		return status, nil
	default:
		return "", domain.ErroValidacao("status da antecipação deve ser ativa ou estornada")
	}
}

func StatusAntecipacaoDe(valor string) StatusAntecipacao {
	return StatusAntecipacao(valor)
}

func (s StatusAntecipacao) String() string {
	return string(s)
}

type AntecipacaoTitulo struct {
	AntecipacaoID  uuid.UUID
	TituloID       uuid.UUID
	ValorCentavos  int64
	StatusAnterior Status
	Vigente        bool
}

type Antecipacao struct {
	ID                   uuid.UUID
	EmpresaID            uuid.UUID
	FornecedorID         uuid.UUID
	Numero               *string
	OperadoEm            time.Time
	TaxaBP               int
	ValorBrutoCentavos   int64
	ValorLiquidoCentavos int64
	ContaBancariaID      uuid.UUID
	Status               StatusAntecipacao
	EstornoMotivo        *string
	EstornadoEm          *time.Time
	UsuarioID            uuid.UUID
	Versao               int
	CriadoEm             time.Time
	AtualizadoEm         time.Time
	Titulos              []*AntecipacaoTitulo
}

type DadosAntecipacao struct {
	FornecedorID         uuid.UUID
	Numero               *string
	OperadoEm            time.Time
	TaxaBP               int
	ValorLiquidoCentavos int64
	ContaBancariaID      uuid.UUID
}

func ErroAntecipacaoNaoEncontrada() error {
	return domain.ErroNaoEncontrado("antecipação não encontrada")
}

func ErroAntecipacaoAlterada() error {
	return domain.ErroConflito("a antecipação foi alterada por outra pessoa; recarregue e tente de novo")
}

func NovaAntecipacao(empresaID, usuarioID uuid.UUID, dados DadosAntecipacao, titulos []*Titulo, agora time.Time) (*Antecipacao, []*Historico, error) {
	switch {
	case empresaID == uuid.Nil:
		return nil, nil, domain.ErroValidacao("empresa da antecipação é obrigatória")
	case usuarioID == uuid.Nil:
		return nil, nil, domain.ErroValidacao("usuário da antecipação não informado")
	case dados.FornecedorID == uuid.Nil:
		return nil, nil, domain.ErroValidacao("fomento da antecipação é obrigatório")
	case dados.ContaBancariaID == uuid.Nil:
		return nil, nil, domain.ErroValidacao("conta bancária do crédito da antecipação é obrigatória")
	case dados.TaxaBP < 0 || dados.TaxaBP > taxaMaximaPontosBase:
		return nil, nil, domain.ErroValidacao("taxa da antecipação deve estar entre 0 e 10000 pontos-base")
	case len(titulos) == 0:
		return nil, nil, domain.ErroValidacao("a antecipação precisa de pelo menos um título")
	}

	operadoEm, err := dataObrigatoria(dados.OperadoEm, feminino("data da operação"))
	if err != nil {
		return nil, nil, err
	}
	if operadoEm.After(Hoje(agora)) {
		return nil, nil, domain.ErroValidacao("data da operação não pode estar no futuro")
	}
	numero, err := numeroOpcional(dados.Numero)
	if err != nil {
		return nil, nil, err
	}

	antecipacao := &Antecipacao{
		ID:                   uuid.New(),
		EmpresaID:            empresaID,
		FornecedorID:         dados.FornecedorID,
		Numero:               numero,
		OperadoEm:            operadoEm,
		TaxaBP:               dados.TaxaBP,
		ValorLiquidoCentavos: dados.ValorLiquidoCentavos,
		ContaBancariaID:      dados.ContaBancariaID,
		Status:               StatusAntecipacaoAtiva,
		UsuarioID:            usuarioID,
		Versao:               versaoInicial,
		CriadoEm:             agora,
		AtualizadoEm:         agora,
		Titulos:              make([]*AntecipacaoTitulo, 0, len(titulos)),
	}

	vistos := make(map[uuid.UUID]struct{}, len(titulos))
	for _, titulo := range titulos {
		if _, repetido := vistos[titulo.ID]; repetido {
			return nil, nil, domain.ErroValidacao("título repetido na antecipação: " + titulo.Numero)
		}
		vistos[titulo.ID] = struct{}{}
		if err := titulo.podeSerAntecipado(empresaID); err != nil {
			return nil, nil, err
		}
		antecipacao.ValorBrutoCentavos += titulo.SaldoCentavos()
	}
	if dados.ValorLiquidoCentavos <= 0 || dados.ValorLiquidoCentavos > antecipacao.ValorBrutoCentavos {
		return nil, nil, domain.ErroValidacao(fmt.Sprintf("valor líquido deve ser maior que zero e no máximo o bruto (%d centavos)", antecipacao.ValorBrutoCentavos))
	}

	historicos := make([]*Historico, 0, len(titulos))
	descricao := "borderô " + antecipacao.descricao()
	for _, titulo := range titulos {
		anterior := titulo.Status
		antecipacao.Titulos = append(antecipacao.Titulos, &AntecipacaoTitulo{
			AntecipacaoID:  antecipacao.ID,
			TituloID:       titulo.ID,
			ValorCentavos:  titulo.SaldoCentavos(),
			StatusAnterior: anterior,
			Vigente:        true,
		})
		titulo.Status = StatusAntecipado
		titulo.AtualizadoEm = agora
		historicos = append(historicos, NovoHistorico(titulo.ID, usuarioID, EventoAntecipado, &anterior, titulo.Status, descricao, agora))
	}

	return antecipacao, historicos, nil
}

func (t *Titulo) podeSerAntecipado(empresaID uuid.UUID) error {
	switch {
	case t.EmpresaID != empresaID:
		return domain.ErroValidacao("título " + t.Numero + " é de outra empresa")
	case !t.Status.NaEmpresa():
		return domain.ErroConflito("só título em carteira ou em banco é antecipado: " + t.Numero)
	case t.SaldoCentavos() <= 0:
		return domain.ErroConflito("título sem saldo não é antecipado: " + t.Numero)
	case t.OrigemCanceladaEm != nil:
		return domain.ErroConflito("título com a origem cancelada não é antecipado: " + t.Numero)
	}

	return nil
}

func (a *Antecipacao) EncargosCentavos() int64 {
	return a.ValorBrutoCentavos - a.ValorLiquidoCentavos
}

func (a *Antecipacao) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão da antecipação é obrigatória")
	}
	if informada != a.Versao {
		return ErroAntecipacaoAlterada()
	}

	return nil
}

func (a *Antecipacao) Vinculo(tituloID uuid.UUID) *AntecipacaoTitulo {
	for _, vinculo := range a.Titulos {
		if vinculo.TituloID == tituloID {
			return vinculo
		}
	}

	return nil
}

func (a *Antecipacao) Estornar(motivo string, titulos map[uuid.UUID]*Titulo, usuarioID uuid.UUID, agora time.Time) ([]*Historico, error) {
	if a.Status != StatusAntecipacaoAtiva {
		return nil, domain.ErroConflito("a antecipação já foi estornada")
	}

	motivo, err := textoObrigatorio(motivo, masculino("motivo do estorno da antecipação"), tamanhoMaximoMotivoEstornoBordero)
	if err != nil {
		return nil, err
	}
	for _, vinculo := range a.Titulos {
		titulo := titulos[vinculo.TituloID]
		if !vinculo.Vigente || titulo == nil || titulo.Status != StatusAntecipado || titulo.SaldoCentavos() != vinculo.ValorCentavos {
			return nil, domain.ErroConflito("a antecipação tem título recebido ou recomprado e não pode ser estornada")
		}
	}

	historicos := make([]*Historico, 0, len(a.Titulos))
	for _, vinculo := range a.Titulos {
		titulo := titulos[vinculo.TituloID]
		anterior := titulo.Status
		titulo.Status = vinculo.StatusAnterior
		titulo.AtualizadoEm = agora
		vinculo.Vigente = false
		historicos = append(historicos, NovoHistorico(titulo.ID, usuarioID, EventoAntecipacaoEstornada, &anterior, titulo.Status,
			"borderô "+a.descricao()+": "+motivo, agora))
	}

	momento := agora
	a.Status = StatusAntecipacaoEstornada
	a.EstornoMotivo = &motivo
	a.EstornadoEm = &momento
	a.AtualizadoEm = agora

	return historicos, nil
}

func (a *Antecipacao) descricao() string {
	if a.Numero != nil {
		return *a.Numero
	}

	return a.OperadoEm.Format(formatoData)
}

func numeroOpcional(valor *string) (*string, error) {
	if valor == nil || strings.TrimSpace(*valor) == "" {
		return nil, nil
	}

	numero, err := textoLimitado(*valor, "número do borderô", tamanhoMaximoNumeroBordero)
	if err != nil {
		return nil, err
	}

	return &numero, nil
}
