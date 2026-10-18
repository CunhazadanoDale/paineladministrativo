package receber

import (
	"fmt"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	versaoInicial                  = 1
	tamanhoMaximoNumero            = 30
	parcelaMaxima                  = 999
	tamanhoMaximoObservacao        = 1000
	tamanhoMaximoMotivoCancelado   = 1000
	tamanhoMaximoMotivoProrrogacao = 500
	formatoData                    = "2006-01-02"
)

type Titulo struct {
	ID                    uuid.UUID
	EmpresaID             uuid.UUID
	ClienteID             uuid.UUID
	Origem                Origem
	OrigemID              *uuid.UUID
	Numero                string
	Parcela               int
	Vencimento            time.Time
	ValorCentavos         int64
	CategoriaFinanceiraID *uuid.UUID
	Competencia           Competencia
	ContaBancariaID       *uuid.UUID
	Status                Status
	RecebidoCentavos      int64
	BoletoEnviadoEm       *time.Time
	BoletoCanal           *CanalBoleto
	OrigemCanceladaEm     *time.Time
	CanceladoMotivo       *string
	Observacao            string
	UsuarioID             uuid.UUID
	Versao                int
	CriadoEm              time.Time
	AtualizadoEm          time.Time
}

type DadosEdicao struct {
	Numero                string
	Parcela               int
	Vencimento            time.Time
	ValorCentavos         int64
	CategoriaFinanceiraID *uuid.UUID
	Competencia           *Competencia
	Observacao            string
}

type DadosTitulo struct {
	DadosEdicao
	ClienteID       uuid.UUID
	ContaBancariaID *uuid.UUID
}

type dadosValidados struct {
	numero      string
	parcela     int
	vencimento  time.Time
	valor       int64
	categoriaID *uuid.UUID
	competencia Competencia
	observacao  string
}

func NovoTituloManual(empresaID, usuarioID uuid.UUID, dados DadosTitulo, agora time.Time) (*Titulo, error) {
	if empresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa do título é obrigatória")
	}
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário do título não informado")
	}
	if dados.ClienteID == uuid.Nil {
		return nil, domain.ErroValidacao("cliente do título é obrigatório")
	}

	validados, err := validarDados(dados.DadosEdicao)
	if err != nil {
		return nil, err
	}
	if validados.vencimento.Before(Hoje(agora)) {
		return nil, erroVencimentoNoPassado()
	}

	titulo := &Titulo{
		ID:              uuid.New(),
		EmpresaID:       empresaID,
		ClienteID:       dados.ClienteID,
		Origem:          OrigemManual,
		ContaBancariaID: idOpcional(dados.ContaBancariaID),
		Status:          StatusCarteira,
		UsuarioID:       usuarioID,
		Versao:          versaoInicial,
		CriadoEm:        agora,
		AtualizadoEm:    agora,
	}
	if titulo.ContaBancariaID != nil {
		titulo.Status = StatusEmBanco
	}
	titulo.aplicar(validados)

	return titulo, nil
}

func validarDados(dados DadosEdicao) (dadosValidados, error) {
	numero, err := textoObrigatorio(dados.Numero, masculino("número do título"), tamanhoMaximoNumero)
	if err != nil {
		return dadosValidados{}, err
	}
	if dados.Parcela < 1 || dados.Parcela > parcelaMaxima {
		return dadosValidados{}, domain.ErroValidacao(fmt.Sprintf("parcela deve estar entre 1 e %d", parcelaMaxima))
	}
	vencimento, err := dataObrigatoria(dados.Vencimento, masculino("vencimento"))
	if err != nil {
		return dadosValidados{}, err
	}
	if dados.ValorCentavos <= 0 {
		return dadosValidados{}, domain.ErroValidacao("valor do título deve ser maior que zero")
	}
	categoriaID := idOpcional(dados.CategoriaFinanceiraID)
	if categoriaID == nil {
		return dadosValidados{}, domain.ErroValidacao("categoria financeira do título manual é obrigatória")
	}
	observacao, err := textoLimitado(dados.Observacao, "observação", tamanhoMaximoObservacao)
	if err != nil {
		return dadosValidados{}, err
	}

	competencia := CompetenciaDe(vencimento)
	if dados.Competencia != nil {
		competencia = *dados.Competencia
	}

	return dadosValidados{
		numero:      numero,
		parcela:     dados.Parcela,
		vencimento:  vencimento,
		valor:       dados.ValorCentavos,
		categoriaID: categoriaID,
		competencia: competencia,
		observacao:  observacao,
	}, nil
}

func (t *Titulo) aplicar(dados dadosValidados) {
	t.Numero = dados.numero
	t.Parcela = dados.parcela
	t.Vencimento = dados.vencimento
	t.ValorCentavos = dados.valor
	t.CategoriaFinanceiraID = dados.categoriaID
	t.Competencia = dados.competencia
	t.Observacao = dados.observacao
}

func ErroVersaoDivergente() error {
	return domain.ErroConflito("o título foi alterado por outra pessoa; recarregue e tente de novo")
}

func (t *Titulo) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão do título é obrigatória")
	}
	if informada != t.Versao {
		return ErroVersaoDivergente()
	}

	return nil
}

func (t *Titulo) SaldoCentavos() int64 {
	return t.ValorCentavos - t.RecebidoCentavos
}

func (t *Titulo) Vencido(hoje time.Time) bool {
	return t.Status.EmAberto() && t.SaldoCentavos() > 0 && t.Vencimento.Before(hoje)
}

func (t *Titulo) PodeSerEditado() error {
	switch {
	case t.Origem != OrigemManual:
		return domain.ErroConflito("só o título manual pode ser editado; o gerado muda pela prorrogação e pela conta de cobrança")
	case t.Status != StatusCarteira:
		return domain.ErroConflito("só o título em carteira pode ser editado")
	case t.RecebidoCentavos > 0:
		return domain.ErroConflito("título com recebimento não pode ser editado")
	}

	return nil
}

func (t *Titulo) Editar(dados DadosEdicao, agora time.Time) (string, error) {
	if err := t.PodeSerEditado(); err != nil {
		return "", err
	}

	validados, err := validarDados(dados)
	if err != nil {
		return "", err
	}
	if !validados.vencimento.Equal(t.Vencimento) && validados.vencimento.Before(Hoje(agora)) {
		return "", erroVencimentoNoPassado()
	}

	alteracoes := t.alteracoes(validados)
	if len(alteracoes) == 0 {
		return "", nil
	}

	t.aplicar(validados)
	t.AtualizadoEm = agora

	return "alterados: " + strings.Join(alteracoes, ", "), nil
}

func (t *Titulo) alteracoes(dados dadosValidados) []string {
	alteracoes := make([]string, 0, 7)
	if dados.numero != t.Numero {
		alteracoes = append(alteracoes, "número")
	}
	if dados.parcela != t.Parcela {
		alteracoes = append(alteracoes, "parcela")
	}
	if !dados.vencimento.Equal(t.Vencimento) {
		alteracoes = append(alteracoes, "vencimento")
	}
	if dados.valor != t.ValorCentavos {
		alteracoes = append(alteracoes, "valor")
	}
	if !mesmoID(dados.categoriaID, t.CategoriaFinanceiraID) {
		alteracoes = append(alteracoes, "categoria")
	}
	if !dados.competencia.Inicio().Equal(t.Competencia.Inicio()) {
		alteracoes = append(alteracoes, "competência")
	}
	if dados.observacao != t.Observacao {
		alteracoes = append(alteracoes, "observação")
	}

	return alteracoes
}

func (t *Titulo) Prorrogar(vencimento time.Time, motivo string, agora time.Time) (string, error) {
	if !t.Status.NaEmpresa() {
		return "", domain.ErroConflito("só o título em carteira ou em banco pode ser prorrogado")
	}

	novo, err := dataObrigatoria(vencimento, masculino("novo vencimento"))
	if err != nil {
		return "", err
	}
	motivo, err = textoObrigatorio(motivo, masculino("motivo da prorrogação"), tamanhoMaximoMotivoProrrogacao)
	if err != nil {
		return "", err
	}
	if !novo.After(t.Vencimento) {
		return "", domain.ErroValidacao("novo vencimento deve ser posterior ao atual")
	}
	if novo.Before(Hoje(agora)) {
		return "", erroVencimentoNoPassado()
	}

	descricao := fmt.Sprintf("vencimento de %s para %s: %s", t.Vencimento.Format(formatoData), novo.Format(formatoData), motivo)
	t.Vencimento = novo
	t.AtualizadoEm = agora

	return descricao, nil
}

func (t *Titulo) EnviarAoBanco(contaBancariaID uuid.UUID, agora time.Time) error {
	if t.Status != StatusCarteira {
		return erroTransicao(t.Status, StatusEmBanco)
	}
	if t.OrigemCanceladaEm != nil {
		return erroOrigemCancelada()
	}
	if contaBancariaID == uuid.Nil {
		return domain.ErroValidacao("conta bancária de cobrança é obrigatória")
	}

	t.ContaBancariaID = &contaBancariaID
	t.Status = StatusEmBanco
	t.AtualizadoEm = agora

	return nil
}

func (t *Titulo) RetirarDoBanco(agora time.Time) error {
	if t.Status != StatusEmBanco {
		return erroTransicao(t.Status, StatusCarteira)
	}

	t.ContaBancariaID = nil
	t.Status = StatusCarteira
	t.AtualizadoEm = agora

	return nil
}

func (t *Titulo) RegistrarEnvioBoleto(canalInformado string, enviadoEm, agora time.Time) error {
	if t.Status != StatusEmBanco {
		return domain.ErroConflito("só o título em banco tem boleto para enviar")
	}

	canal, err := NovoCanalBoleto(canalInformado)
	if err != nil {
		return err
	}
	momento, err := momentoDoFato(enviadoEm, agora, masculino("envio do boleto"))
	if err != nil {
		return err
	}

	t.BoletoCanal = &canal
	t.BoletoEnviadoEm = &momento
	t.AtualizadoEm = agora

	return nil
}

func (t *Titulo) Cancelar(motivo string, agora time.Time) error {
	if !t.Status.NaEmpresa() {
		return erroTransicao(t.Status, StatusCancelado)
	}
	if t.RecebidoCentavos > 0 {
		return domain.ErroConflito("título com recebimento não pode ser cancelado; estorne o recebimento antes")
	}

	motivo, err := textoObrigatorio(motivo, masculino("motivo do cancelamento"), tamanhoMaximoMotivoCancelado)
	if err != nil {
		return err
	}

	t.CanceladoMotivo = &motivo
	t.Status = StatusCancelado
	t.AtualizadoEm = agora

	return nil
}

func erroTransicao(atual, proximo Status) error {
	return domain.ErroConflito(fmt.Sprintf("não é possível mudar o título de %q para %q", atual, proximo))
}

func erroVencimentoNoPassado() error {
	return domain.ErroValidacao("vencimento não pode ser anterior a hoje")
}

func erroOrigemCancelada() error {
	return domain.ErroConflito("a origem do título foi cancelada; resolva o título antes de movimentá-lo")
}
