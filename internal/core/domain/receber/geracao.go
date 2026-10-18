package receber

import (
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoMotivoDaOrigem = 1000

type ParcelaDeOrigem struct {
	OrigemID      uuid.UUID
	Numero        string
	Parcela       int
	Vencimento    time.Time
	ValorCentavos int64
}

type Geracao struct {
	Origem     Origem
	EmpresaID  uuid.UUID
	ClienteID  uuid.UUID
	UsuarioID  uuid.UUID
	Titulos    []*Titulo
	Historicos []*Historico
}

type ResultadoGeracao struct {
	Gerados      []uuid.UUID
	JaExistentes []uuid.UUID
}

func NovaGeracao(
	origem Origem,
	empresaID, clienteID uuid.UUID,
	contaBancariaID *uuid.UUID,
	parcelas []ParcelaDeOrigem,
	usuarioID uuid.UUID,
	agora time.Time,
) (*Geracao, error) {
	switch {
	case !origem.Gerada():
		return nil, domain.ErroValidacao("a geração de títulos exige uma origem diferente de manual")
	case origem != OrigemPedido && origem != OrigemNotaFiscalDuplicata:
		return nil, domain.ErroValidacao("origem do título inválida")
	case empresaID == uuid.Nil:
		return nil, ErroEmpresaObrigatoria()
	case clienteID == uuid.Nil:
		return nil, ErroClienteObrigatorio()
	case usuarioID == uuid.Nil:
		return nil, domain.ErroValidacao("usuário da geração não informado")
	case len(parcelas) == 0:
		return nil, domain.ErroValidacao("a geração precisa de pelo menos uma parcela")
	}

	conta := idOpcional(contaBancariaID)
	geracao := &Geracao{
		Origem:     origem,
		EmpresaID:  empresaID,
		ClienteID:  clienteID,
		UsuarioID:  usuarioID,
		Titulos:    make([]*Titulo, 0, len(parcelas)),
		Historicos: make([]*Historico, 0, len(parcelas)),
	}
	vistas := make(map[uuid.UUID]struct{}, len(parcelas))
	for indice, parcela := range parcelas {
		if parcela.OrigemID == uuid.Nil {
			return nil, domain.ErroValidacao(fmt.Sprintf("parcela %d sem o id da origem", indice+1))
		}
		if _, repetida := vistas[parcela.OrigemID]; repetida {
			return nil, domain.ErroValidacao(fmt.Sprintf("parcela %d repete o id da origem", indice+1))
		}
		vistas[parcela.OrigemID] = struct{}{}

		titulo, err := novoTituloDeOrigem(origem, empresaID, clienteID, conta, parcela, usuarioID, agora)
		if err != nil {
			return nil, err
		}
		geracao.Titulos = append(geracao.Titulos, titulo)
		geracao.Historicos = append(geracao.Historicos, NovoHistorico(titulo.ID, usuarioID, EventoGerado, nil, titulo.Status, "", agora))
	}

	return geracao, nil
}

func (g *Geracao) OrigemIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(g.Titulos))
	for _, titulo := range g.Titulos {
		ids = append(ids, *titulo.OrigemID)
	}

	return ids
}

func novoTituloDeOrigem(
	origem Origem,
	empresaID, clienteID uuid.UUID,
	contaBancariaID *uuid.UUID,
	parcela ParcelaDeOrigem,
	usuarioID uuid.UUID,
	agora time.Time,
) (*Titulo, error) {
	numero, err := textoObrigatorio(parcela.Numero, masculino("número do título"), tamanhoMaximoNumero)
	if err != nil {
		return nil, err
	}
	if parcela.Parcela < 1 || parcela.Parcela > parcelaMaxima {
		return nil, domain.ErroValidacao(fmt.Sprintf("parcela deve estar entre 1 e %d", parcelaMaxima))
	}
	vencimento, err := dataObrigatoria(parcela.Vencimento, masculino("vencimento"))
	if err != nil {
		return nil, err
	}
	if parcela.ValorCentavos <= 0 {
		return nil, domain.ErroValidacao("valor do título deve ser maior que zero")
	}

	origemID := parcela.OrigemID
	titulo := &Titulo{
		ID:              uuid.New(),
		EmpresaID:       empresaID,
		ClienteID:       clienteID,
		Origem:          origem,
		OrigemID:        &origemID,
		Numero:          numero,
		Parcela:         parcela.Parcela,
		Vencimento:      vencimento,
		ValorCentavos:   parcela.ValorCentavos,
		Competencia:     CompetenciaDe(vencimento),
		ContaBancariaID: idOpcional(contaBancariaID),
		Status:          StatusCarteira,
		UsuarioID:       usuarioID,
		Versao:          versaoInicial,
		CriadoEm:        agora,
		AtualizadoEm:    agora,
	}
	if titulo.ContaBancariaID != nil {
		titulo.Status = StatusEmBanco
	}

	return titulo, nil
}

type CancelamentoDeOrigem struct {
	Origem    Origem
	OrigemIDs []uuid.UUID
	Motivo    string
	UsuarioID uuid.UUID
	Agora     time.Time
}

type ResultadoCancelamento struct {
	Cancelados []uuid.UUID
	Marcados   []uuid.UUID
	Ignorados  int
}

func NovoCancelamentoDeOrigem(origem Origem, origemIDs []uuid.UUID, motivo string, usuarioID uuid.UUID, agora time.Time) (CancelamentoDeOrigem, error) {
	switch {
	case !origem.Gerada():
		return CancelamentoDeOrigem{}, domain.ErroValidacao("o cancelamento por origem exige uma origem diferente de manual")
	case usuarioID == uuid.Nil:
		return CancelamentoDeOrigem{}, domain.ErroValidacao("usuário do cancelamento não informado")
	case len(origemIDs) == 0:
		return CancelamentoDeOrigem{}, domain.ErroValidacao("o cancelamento precisa de pelo menos um id de origem")
	}

	motivo, err := textoObrigatorio(motivo, masculino("motivo do cancelamento da origem"), tamanhoMaximoMotivoDaOrigem)
	if err != nil {
		return CancelamentoDeOrigem{}, err
	}

	ids := make([]uuid.UUID, 0, len(origemIDs))
	vistas := make(map[uuid.UUID]struct{}, len(origemIDs))
	for _, id := range origemIDs {
		if id == uuid.Nil {
			return CancelamentoDeOrigem{}, domain.ErroValidacao("id de origem vazio no cancelamento")
		}
		if _, repetido := vistas[id]; repetido {
			continue
		}
		vistas[id] = struct{}{}
		ids = append(ids, id)
	}

	return CancelamentoDeOrigem{Origem: origem, OrigemIDs: ids, Motivo: motivo, UsuarioID: usuarioID, Agora: agora}, nil
}

type EfeitoDaOrigemCancelada string

const (
	EfeitoCancelado EfeitoDaOrigemCancelada = "cancelado"
	EfeitoMarcado   EfeitoDaOrigemCancelada = "marcado"
	EfeitoIgnorado  EfeitoDaOrigemCancelada = "ignorado"
)

func (t *Titulo) AplicarCancelamentoDaOrigem(cancelamento CancelamentoDeOrigem) (EfeitoDaOrigemCancelada, *Historico) {
	anterior := t.Status

	switch {
	case t.Status == StatusCancelado:
		return EfeitoIgnorado, nil
	case t.Status.NaEmpresa() && t.RecebidoCentavos == 0:
		motivo := cancelamento.Motivo
		t.CanceladoMotivo = &motivo
		t.Status = StatusCancelado
		t.AtualizadoEm = cancelamento.Agora
		return EfeitoCancelado, NovoHistorico(t.ID, cancelamento.UsuarioID, EventoCancelado, &anterior, t.Status, motivo, cancelamento.Agora)
	case t.OrigemCanceladaEm != nil:
		return EfeitoIgnorado, nil
	default:
		momento := cancelamento.Agora
		t.OrigemCanceladaEm = &momento
		t.AtualizadoEm = cancelamento.Agora
		return EfeitoMarcado, NovoHistorico(t.ID, cancelamento.UsuarioID, EventoOrigemCancelada, &anterior, t.Status, cancelamento.Motivo, cancelamento.Agora)
	}
}
