package caixa

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	tamanhoMaximoDescricao = 255
	tamanhoMaximoDocumento = 60
	descricaoAusente       = "Sem descrição"
)

type MotivoIgnorada string

const (
	MotivoAnteriorAoCorte MotivoIgnorada = "anterior_ao_saldo_inicial"
	MotivoDataFutura      MotivoIgnorada = "data_futura"
	MotivoValorZero       MotivoIgnorada = "valor_zero"
	MotivoMesFechado      MotivoIgnorada = "mes_fechado"
)

func (m MotivoIgnorada) String() string {
	return string(m)
}

type ImportacaoExtrato struct {
	ID                 uuid.UUID
	ContaBancariaID    uuid.UUID
	ArquivoID          uuid.UUID
	PeriodoInicio      time.Time
	PeriodoFim         time.Time
	SaldoBancoCentavos *int64
	SaldoBancoEm       *time.Time
	LinhasNovas        int
	LinhasRepetidas    int
	LinhasIgnoradas    int
	UsuarioID          uuid.UUID
	CriadoEm           time.Time
}

type LancamentoExtrato struct {
	ID              uuid.UUID
	ContaBancariaID uuid.UUID
	ImportacaoID    uuid.UUID
	Sequencia       int
	Data            time.Time
	ValorCentavos   int64
	Descricao       string
	Documento       *string
	Identificador   string
	CriadoEm        time.Time
	ConciliacaoID   *uuid.UUID
	Situacao        SituacaoLancamento
}

func (l *LancamentoExtrato) Credito() bool {
	return l.ValorCentavos > 0
}

type LinhaIgnorada struct {
	Sequencia     int
	Data          time.Time
	ValorCentavos int64
	Descricao     string
	Motivo        MotivoIgnorada
}

type DadosImportacao struct {
	Extrato         *ExtratoOFX
	ContaBancariaID uuid.UUID
	ArquivoID       uuid.UUID
	UsuarioID       uuid.UUID
	Abertura        *Abertura
	MesesFechados   []time.Time
	Agora           time.Time
}

type ExtratoPreparado struct {
	Importacao  *ImportacaoExtrato
	Lancamentos []*LancamentoExtrato
	Ignoradas   []LinhaIgnorada
}

func PrepararImportacao(dados DadosImportacao) (*ExtratoPreparado, error) {
	if dados.Extrato == nil {
		return nil, domain.ErroValidacao("extrato não informado")
	}
	if dados.ContaBancariaID == uuid.Nil || dados.ArquivoID == uuid.Nil || dados.UsuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("conta, arquivo e usuário da importação são obrigatórios")
	}
	if dados.Abertura == nil || dados.Abertura.ContaBancariaID != dados.ContaBancariaID {
		return nil, ErroContaSemAbertura()
	}

	inicio, fim, err := periodoDoExtrato(dados.Extrato)
	if err != nil {
		return nil, err
	}

	importacao := &ImportacaoExtrato{
		ID:              uuid.New(),
		ContaBancariaID: dados.ContaBancariaID,
		ArquivoID:       dados.ArquivoID,
		PeriodoInicio:   inicio,
		PeriodoFim:      fim,
		UsuarioID:       dados.UsuarioID,
		CriadoEm:        dados.Agora,
	}
	if saldo := dados.Extrato.Saldo; saldo != nil {
		centavos, em := saldo.Centavos, DataDe(saldo.Em)
		importacao.SaldoBancoCentavos = &centavos
		importacao.SaldoBancoEm = &em
	}

	preparado := &ExtratoPreparado{Importacao: importacao}
	hoje := Hoje(dados.Agora)
	ocorrencias := map[string]int{}
	fechados := make(map[time.Time]bool, len(dados.MesesFechados))
	for _, competencia := range dados.MesesFechados {
		fechados[CompetenciaDe(competencia)] = true
	}

	for i, transacao := range dados.Extrato.Transacoes {
		sequencia := i + 1
		data := DataDe(transacao.Data)
		descricao := textoLimitado(transacao.Descricao, tamanhoMaximoDescricao)
		if descricao == "" {
			descricao = descricaoAusente
		}

		chave := chaveDaTransacao(transacao, data, descricao)
		ocorrencias[chave]++

		if motivo, ignorar := motivoParaIgnorar(data, transacao.ValorCentavos, dados.Abertura, fechados, hoje); ignorar {
			preparado.Ignoradas = append(preparado.Ignoradas, LinhaIgnorada{
				Sequencia:     sequencia,
				Data:          data,
				ValorCentavos: transacao.ValorCentavos,
				Descricao:     descricao,
				Motivo:        motivo,
			})
			continue
		}

		preparado.Lancamentos = append(preparado.Lancamentos, &LancamentoExtrato{
			ID:              uuid.New(),
			ContaBancariaID: dados.ContaBancariaID,
			ImportacaoID:    importacao.ID,
			Sequencia:       sequencia,
			Data:            data,
			ValorCentavos:   transacao.ValorCentavos,
			Descricao:       descricao,
			Documento:       documentoDaTransacao(transacao.Documento),
			Identificador:   identificador(chave, ocorrencias[chave]),
			CriadoEm:        dados.Agora,
			Situacao:        LancamentoPendente,
		})
	}

	importacao.LinhasIgnoradas = len(preparado.Ignoradas)

	return preparado, nil
}

func (p *ExtratoPreparado) Total() int {
	return len(p.Lancamentos) + len(p.Ignoradas)
}

func (p *ExtratoPreparado) RegistrarNovas(novas int) {
	p.Importacao.LinhasNovas = novas
	p.Importacao.LinhasRepetidas = len(p.Lancamentos) - novas
}

func (p *ExtratoPreparado) Identificadores() []string {
	identificadores := make([]string, 0, len(p.Lancamentos))
	for _, lancamento := range p.Lancamentos {
		identificadores = append(identificadores, lancamento.Identificador)
	}

	return identificadores
}

func periodoDoExtrato(extrato *ExtratoOFX) (time.Time, time.Time, error) {
	var inicio, fim time.Time
	for _, transacao := range extrato.Transacoes {
		data := DataDe(transacao.Data)
		if inicio.IsZero() || data.Before(inicio) {
			inicio = data
		}
		if fim.IsZero() || data.After(fim) {
			fim = data
		}
	}

	if extrato.Inicio != nil && (inicio.IsZero() || DataDe(*extrato.Inicio).Before(inicio)) {
		inicio = DataDe(*extrato.Inicio)
	}
	if extrato.Fim != nil && (fim.IsZero() || DataDe(*extrato.Fim).After(fim)) {
		fim = DataDe(*extrato.Fim)
	}

	if inicio.IsZero() || fim.IsZero() {
		return time.Time{}, time.Time{}, domain.ErroValidacao("extrato sem lançamentos nem período")
	}
	if fim.Before(inicio) {
		return time.Time{}, time.Time{}, domain.ErroValidacao("período do extrato inválido")
	}

	return inicio, fim, nil
}

func motivoParaIgnorar(data time.Time, valor int64, abertura *Abertura, fechados map[time.Time]bool, hoje time.Time) (MotivoIgnorada, bool) {
	switch {
	case valor == 0:
		return MotivoValorZero, true
	case !abertura.Abrange(data):
		return MotivoAnteriorAoCorte, true
	case fechados[CompetenciaDe(data)]:
		return MotivoMesFechado, true
	case data.After(hoje):
		return MotivoDataFutura, true
	default:
		return "", false
	}
}

func chaveDaTransacao(transacao TransacaoOFX, data time.Time, descricao string) string {
	chave := transacao.FITID + "|" + data.Format(formatoDataOFX) + "|" + strconv.FormatInt(transacao.ValorCentavos, 10)
	if transacao.FITID == "" {
		chave += "|" + descricao
	}

	return chave
}

func identificador(chave string, ocorrencia int) string {
	soma := sha256.Sum256([]byte(chave + "|" + strconv.Itoa(ocorrencia)))

	return hex.EncodeToString(soma[:])
}

func documentoDaTransacao(valor string) *string {
	documento := textoLimitado(valor, tamanhoMaximoDocumento)
	if documento == "" {
		return nil
	}

	return &documento
}
