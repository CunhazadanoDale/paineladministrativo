package caixa

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type MovimentoDoMes struct {
	EntradasCentavos        int64
	SaidasCentavos          int64
	LancamentosPendentes    int
	ConciliacoesInvalidadas int
	SaldoBancoCentavos      *int64
}

type PreviaFechamento struct {
	ContaBancariaID      uuid.UUID
	Competencia          time.Time
	SaldoInicialCentavos int64
	MesEncerrado         bool
	MovimentoDoMes
	MovimentosSemLinha []*MovimentoSistema
}

func (p *PreviaFechamento) PodeFechar() bool {
	return p.MesEncerrado && p.Bloqueios() == nil && p.SaldoBancoCentavos != nil &&
		*p.SaldoBancoCentavos == p.SaldoFinalCentavos()
}

func (p *PreviaFechamento) SaldoFinalCentavos() int64 {
	return p.SaldoInicialCentavos + p.EntradasCentavos - p.SaidasCentavos
}

func (p *PreviaFechamento) Bloqueios() error {
	if p.LancamentosPendentes == 0 && p.ConciliacoesInvalidadas == 0 {
		return nil
	}

	return domain.ErroConflito("o mês tem " + strconv.Itoa(p.LancamentosPendentes) + " lançamento(s) pendente(s) e " +
		strconv.Itoa(p.ConciliacoesInvalidadas) + " conciliação(ões) invalidada(s); concilie, classifique ou desfaça antes de fechar")
}

type Fechamento struct {
	ID                   uuid.UUID
	ContaBancariaID      uuid.UUID
	Competencia          time.Time
	SaldoInicialCentavos int64
	EntradasCentavos     int64
	SaidasCentavos       int64
	SaldoFinalCentavos   int64
	SaldoBancoCentavos   int64
	Vigente              bool
	ReabertoEm           *time.Time
	ReabertoPor          *uuid.UUID
	MotivoReabertura     *string
	UsuarioID            uuid.UUID
	CriadoEm             time.Time
}

type DadosFechamento struct {
	Previa              *PreviaFechamento
	SaldoBancoInformado *int64
	UsuarioID           uuid.UUID
	Agora               time.Time
}

func NovoFechamento(dados DadosFechamento) (*Fechamento, error) {
	previa := dados.Previa
	if previa == nil {
		return nil, domain.ErroValidacao("prévia do fechamento não informada")
	}
	if dados.UsuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário não informado")
	}
	if err := previa.Bloqueios(); err != nil {
		return nil, err
	}

	saldoBanco := dados.SaldoBancoInformado
	if saldoBanco == nil {
		saldoBanco = previa.SaldoBancoCentavos
	}
	if saldoBanco == nil {
		return nil, domain.ErroValidacao("informe o saldo do banco no fim do mês; o extrato importado não traz o saldo do último dia")
	}

	saldoFinal := previa.SaldoFinalCentavos()
	if *saldoBanco != saldoFinal {
		return nil, domain.ErroConflito("saldo calculado (" + FormatarCentavos(saldoFinal) + ") diferente do saldo do banco (" +
			FormatarCentavos(*saldoBanco) + "); diferença de " + FormatarCentavos(*saldoBanco-saldoFinal))
	}

	return &Fechamento{
		ID:                   uuid.New(),
		ContaBancariaID:      previa.ContaBancariaID,
		Competencia:          previa.Competencia,
		SaldoInicialCentavos: previa.SaldoInicialCentavos,
		EntradasCentavos:     previa.EntradasCentavos,
		SaidasCentavos:       previa.SaidasCentavos,
		SaldoFinalCentavos:   saldoFinal,
		SaldoBancoCentavos:   *saldoBanco,
		Vigente:              true,
		UsuarioID:            dados.UsuarioID,
		CriadoEm:             dados.Agora,
	}, nil
}

func (f *Fechamento) Reabrir(usuarioID uuid.UUID, motivo string, agora time.Time) error {
	if !f.Vigente {
		return ErroFechamentoJaReaberto()
	}
	if usuarioID == uuid.Nil {
		return domain.ErroValidacao("usuário não informado")
	}

	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return domain.ErroValidacao("motivo da reabertura é obrigatório")
	}
	if utf8.RuneCountInString(motivo) > tamanhoMaximoMotivo {
		return domain.ErroValidacao("motivo deve ter no máximo " + strconv.Itoa(tamanhoMaximoMotivo) + " caracteres")
	}

	f.Vigente = false
	f.ReabertoEm = &agora
	f.ReabertoPor = &usuarioID
	f.MotivoReabertura = &motivo

	return nil
}

func (a *Abertura) PrimeiraCompetencia() time.Time {
	return CompetenciaDe(a.DataCorte.AddDate(0, 0, 1))
}

func ProximaCompetencia(abertura *Abertura, ultimo *Fechamento) time.Time {
	if ultimo != nil {
		return ultimo.Competencia.AddDate(0, 1, 0)
	}

	return abertura.PrimeiraCompetencia()
}

func SaldoInicialDaCompetencia(abertura *Abertura, ultimo *Fechamento) int64 {
	if ultimo != nil {
		return ultimo.SaldoFinalCentavos
	}

	return abertura.SaldoCentavos
}

func ConferirCompetencia(competencia time.Time, abertura *Abertura, ultimo *Fechamento, hoje time.Time, exigirEncerrado bool) error {
	if competencia.IsZero() {
		return domain.ErroValidacao("competência é obrigatória")
	}
	if abertura == nil {
		return ErroContaSemAbertura()
	}

	competencia = CompetenciaDe(competencia)
	if competencia.Before(abertura.PrimeiraCompetencia()) {
		return domain.ErroValidacao("competência anterior ao saldo inicial da conta")
	}
	if exigirEncerrado && !competencia.Before(CompetenciaDe(hoje)) {
		return domain.ErroValidacao("só é possível fechar mês encerrado")
	}

	proxima := ProximaCompetencia(abertura, ultimo)
	if competencia.Before(proxima) {
		return domain.ErroConflito("mês já fechado nesta conta")
	}
	if competencia.After(proxima) {
		return domain.ErroConflito("feche antes o mês de " + proxima.Format("01/2006"))
	}

	return nil
}

func FimDaCompetencia(competencia time.Time) time.Time {
	return CompetenciaDe(competencia).AddDate(0, 1, -1)
}
