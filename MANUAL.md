# Manual do Painel — Grupo MIA / Red Diamond / Gigante

Resumo de como funciona cada página do front e cada funcionalidade do backend, com
exemplos de "como fazer". Situação revisada em 19/10/2026 (T0, F0, M1 a M11 concluídos;
M12 a M15 pendentes; front de comissões em andamento).

- **Parte 1:** conceitos que valem para tudo (perfis, empresas, valores, erros).
- **Parte 2:** frontend, ou seja, o que o usuário vê e faz em cada tela.
- **Parte 3:** backend, ou seja, as regras e endpoints de cada módulo, com exemplos de chamada.
- **Parte 4:** lacunas e inconsistências encontradas na revisão.

---

# Parte 1 — Conceitos gerais

## 1.1 Quem pode o quê

O acesso depende de **três coisas independentes**:

| O quê | Onde se configura | Efeito |
|---|---|---|
| **Cargo** com as caixas Administrador, Financeiro e Comercial (cumulativas) | Sistema → Cargos, e depois Usuários → Editar → Cargo | Administrador passa em tudo. Comercial libera leads, funis, pedidos e clientes. Financeiro libera contas, recebíveis, NF e despesas |
| **Aprovador** | Financeiro → Aprovadores | Ganha a aba "Aprovação" em Solicitações e os botões Aprovar e Rejeitar |
| **Empresas do usuário** | Usuários → ícone de prédio | Define quais CNPJs o usuário enxerga. **Sem nenhum vínculo, ele não vê nada e não abre solicitação** |

O **representante com login** é um caso especial: um usuário (não administrador) vinculado a
um cadastro de Representante. Ele só vê e altera os clientes e pedidos da própria
carteira, nunca define a própria comissão e nunca apura nem fecha comissões.

**Regras que pegam todo mundo de surpresa:**
- Ninguém aprova a própria solicitação, nem o administrador.
- Solicitação sem categoria financeira não pode ser paga.
- O sistema não deixa ficar sem nenhum administrador ativo.

## 1.2 Empresas

Não existe seletor global de empresa. Cada formulário e cada filtro tem o seu próprio
campo "Empresa", que lista só as empresas do acesso do usuário. Quando só há uma, ela já
vem selecionada. Empresa inativa não recebe lançamento novo.

## 1.3 Valores, datas e versões

- **Dinheiro:** na tela, `R$ 1.500,00`. Os campos aceitam `1.500,00`, `1500,5`, `1500` ou
  `R$ 1.500`. Na API o valor vai sempre em centavos inteiros: `150000` = R$ 1.500,00.
- **Percentual:** na API vai em pontos-base. `300` = 3%; `10000` = 100%.
- **Datas:** a data do fato (vencimento, saída, pagamento) é informada pelo usuário e não
  pode estar no futuro quando não fizer sentido. A data de registro é automática.
- **Versão:** quase todo registro tem `versao`. Se duas pessoas editam o mesmo registro, a
  segunda recebe **409** e o front mostra o aviso amarelo **"Recarregar dados"**. Basta
  recarregar, conferir e salvar de novo.
- **Nada se apaga do histórico.** Para desfazer um lançamento faz-se um lançamento
  inverso: estorno de pagamento, estorno de recebimento, estorno de borderô, movimento de
  estoque inverso ou ajuste de comissão.
- **Excluir quase nunca é o caminho.** Cadastro com vínculo só pode ser **desativado**.

## 1.4 Mensagens de erro

| Código | Quando acontece | O que fazer |
|---|---|---|
| 400 | Dado inválido ("erro de validação: …") ou registro em uso | Corrigir o campo indicado |
| 401 | Sessão vencida, senha trocada ou logout em outra aba | Entrar de novo |
| 403 | Perfil sem permissão ou empresa fora do acesso | Pedir acesso ao administrador |
| 404 | Registro não existe | — |
| 409 | Duplicado, status que não permite a ação, ou alteração simultânea | Recarregar e conferir |
| 429 | Muitas senhas erradas no login (o botão vira "Aguarde N s") | Esperar o contador |

---

# Parte 2 — Frontend (telas)

Repo: `D:\PROJETOS\FRONT\painelMiaConstrutora`. As rotas ficam em `src/App.jsx`, o menu em
`src/app/layout/Sidebar.jsx` e as telas em `src/modules/<modulo>`.

## 2.1 Navegação

- **Menu lateral** recolhível (seta no rodapé). Os grupos são Início, Comercial,
  Financeiro, Site & Catálogo e Sistema. Itens fora do perfil do usuário não aparecem. Se
  alguém digitar a URL de uma rota proibida, volta ao Dashboard sem mensagem.
- **Barra superior:**
  - título da página;
  - busca de leads (só para comercial);
  - avatar, que leva a **Meu perfil**;
  - botão **Sair**, que encerra a sessão no servidor e em todas as abas.
- **Contador de Solicitações** no menu: "X para aprovar · Y para pagar · Z vencidas".
- **Páginas provisórias**, que ainda não fazem nada: Negociações, Formulários,
  Relatórios e Configurações.

## 2.2 Login (`/login`)

Informe e-mail e senha e clique em **Entrar**. A sessão dura 8 horas, sem renovação
automática. Depois de 5 erros seguidos o botão fica bloqueado por alguns segundos.

## 2.3 Meu perfil (`/perfil`)

Mostra nome, e-mail, cargo e o último acesso.

**Como trocar a própria senha:** Avatar → Trocar senha → Senha atual, Nova senha
(mínimo 8 caracteres) e Confirmação → Salvar. A sessão é encerrada e é preciso entrar de
novo com a senha nova.

## 2.4 Dashboard (`/`)

- **Quem não é comercial:** vê apenas a saudação.
- **Quem é comercial:** vê os cartões (total de leads, funis ativos, etapas, funil com
  mais leads), os gráficos de leads por funil e por etapa, e os 8 últimos leads.

---

## 2.5 Sistema (só administrador)

### Empresas (`/empresas`)

Lista os CNPJs do grupo. Os botões são Nova Empresa, editar, ativar/desativar e excluir.

**Como cadastrar uma filial:**
1. Abra Empresas e clique em **Nova Empresa**.
2. Preencha:
   - CNPJ (com ou sem máscara);
   - Razão social;
   - Nome fantasia;
   - UF;
   - Regime tributário (Simples, Presumido ou Real).
3. Salve.
4. Dê acesso aos usuários em **Usuários → ícone de prédio**.

> O CNPJ não pode ser alterado. Se estiver errado, desative a empresa e cadastre outra.
> Empresa com vínculo não pode ser excluída, só desativada.

### Cargos (`/cargos`)

Nome, descrição e as caixas **Administrador**, **Financeiro** e **Comercial**.

**Como criar um perfil "Financeiro":**
1. Clique em **Novo Cargo**.
2. Nome: "Financeiro". Marque só a caixa **Financeiro**.
3. Salve.
4. Atribua o cargo em Usuários → Editar → Cargo.

Cargo com usuários não pode ser excluído.

### Usuários (`/usuarios`)

Busca por nome ou e-mail. Ações por linha:

| Ícone | Ação |
|---|---|
| Lápis | Editar nome, e-mail e cargo |
| Chave | Trocar a senha do usuário (derruba a sessão dele) |
| Prédio | Empresas que ele acessa: marque as empresas ou "Acesso a todas" (inclui as futuras) |
| Ativar/Desativar | Bloqueia o login sem apagar nada |
| Lixeira | Excluir (não tem volta) |

**Como cadastrar um usuário novo:**
1. Clique em **Novo Usuário** e preencha Nome, E-mail, Cargo e Senha (8 caracteres ou
   mais).
2. **Clique no prédio** e marque as empresas. Sem este passo o usuário não vê nada.
3. Se ele for aprovar gastos, designe-o em **Financeiro → Aprovadores**.

---

## 2.6 Comercial

### Funis (`/funis`) — comercial vê; só administrador edita

- À esquerda, os funis. À direita, as etapas do funil selecionado, com a quantidade de
  leads em cada uma.
- **Como montar um funil:**
  1. Clique em **Novo Funil** e informe o nome (ex.: "Vendas PB").
  2. Clique em **Nova Etapa** para cada passo: "Novo contato", "Proposta", "Fechado".
  3. Use as setas ↑/↓ para ordenar.
- Funil ou etapa com leads não pode ser excluído: mova ou exclua os leads antes.
- A cor da etapa sai do nome: "perdido" fica vermelho, "proposta" ou "fechado" ficam
  verdes, "contato" fica amarelo e "novo" fica azul.

### Leads (`/leads`) — só comercial

- **Modo Lista:** tabela com filtro por texto e por etapa.
- **Modo Kanban:** uma coluna por etapa. **Arrastar o cartão** muda a etapa e grava o
  histórico.
- **Novo Lead:** Nome (obrigatório), E-mail, Telefone, Origem ("Site", "Indicação") e
  Etapa.
- **Detalhe (`/leads/:id`):** dados, linha do tempo de movimentações e o botão
  **Converter em cliente**.

**Como converter um lead em cliente:**
1. Abra o lead e clique em **Converter em cliente**.
2. O formulário de cliente abre já preenchido. Complete CPF/CNPJ, Tipo, UF e Cidade.
3. Salve. O lead continua no funil e passa a mostrar o link "Cliente: Nome".

### Representantes (`/representantes`) — comercial vê; só administrador edita

Campos: Nome, Documento, **Usuário com login** (opcional), E-mail, Telefone, **Comissão
pronta entrega (%)** e **Comissão importação (%)**.

> Ao ligar um usuário, ele passa a ver **só os clientes e pedidos da própria carteira**.

### Clientes (`/clientes`) e Grupos de Clientes (`/grupos-clientes`)

- **Filtros:** busca por nome ou documento, Tipo, UF, Grupo, Representante e Situação.
- **Novo Cliente** (comercial):
  - obrigatórios: CPF/CNPJ, Tipo (Construtora, Loja, Pessoa física ou Outro), Nome, UF
    e Cidade;
  - opcionais: grupo, representante, empresa de relacionamento, e-mail para NF,
    telefone e observações;
  - regras comerciais: **"Exige número do pedido na NF"** e **"Proíbe antecipação dos
    títulos"**. A segunda bloqueia esse cliente em qualquer borderô.
- **Pessoa física** só aceita CPF, e CPF só aceita o tipo pessoa física. O documento não
  muda depois de salvo.
- **Detalhe do cliente:** dados e a lista de **Obras**. Use **Nova obra** para informar
  nome, CNPJ da SPE (opcional) e endereço com CEP.
- **Importar CSV** (administrador):
  - formato: UTF-8, separado por `;`, até 5.000 linhas;
  - cabeçalho:
    `documento;nome;nome_fantasia;tipo;grupo;uf;cidade;email_nf;telefone;representante_documento;exige_numero_pedido_na_nf;proibe_antecipacao;observacoes`;
  - clique em **Simular** primeiro, confira as linhas rejeitadas e depois clique em
    **Importar**.
- **Grupos:** cadastro só de nome, com renomear e ativar/desativar. Não há exclusão.

**Como cadastrar um cliente com representante:**
1. O administrador cria o representante em Representantes, com os percentuais.
2. Em Clientes, clique em **Novo Cliente** e preencha CNPJ, "Construtora", nome, UF,
   cidade e o representante.
3. Abra o cliente e clique em **Nova obra**.

### Pedidos (`/pedidos`), Entregas (`/entregas`) e Consultas (`/pedidos/consultas`)

**Status do pedido:** `Proposta → Confirmado → Parcialmente entregue → Entregue`; ou
`Cancelado`.

| Modelo | Condição padrão |
|---|---|
| Pronta entrega | Boleto em 30 dias; prazo de entrega digitado (opcional) |
| Importação | Entrada de 30%; entrega em até 120 dias contados da entrada paga |

**Como criar um pedido e entregar:**
1. **Pedidos → Nova Proposta.**
   - Escolha Empresa, Cliente (o representante vem preenchido) e Modelo.
   - Se quiser, escolha a Obra e o Nº do pedido do cliente.
   - Adicione os itens: produto, quantidade e preço. O preço vem sugerido do cadastro do
     produto.
2. Abra o pedido e clique em **Confirmar**.
   - O sistema reserva o estoque disponível nos locais da empresa, começando pelo local
     com mais disponível.
   - O que faltar aparece em "Falta cobrir".
   - Depois de confirmado, os itens não mudam mais.
3. Quando chegar estoque, use **Reservar pendências**.
4. **Importação:** o financeiro clica em **Gerar título de entrada** (30%). Quando esse
   título é recebido por inteiro, a entrada é registrada sozinha e o prazo de 120 dias
   começa a contar. A alternativa é usar **Registrar entrada** à mão.
5. No painel Entregas, clique em **Programar entrega**.
   - Frete CIF (valor obrigatório) ou FOB.
   - Transportadora e previsão.
   - Para cada item, a quantidade e o local de saída.
   - Programar ainda não mexe no estoque.
6. Na linha da entrega, clique em **Dar saída** e informe a data. O estoque **é baixado
   aqui**.
7. Clique em **Marcar entregue** e informe a data.

**Para cancelar:** clique em **Cancelar** e informe o motivo (e o valor a devolver, se
houve entrada). As reservas são liberadas. Entrega em trânsito impede o cancelamento.

**Telas de apoio:**
- **Entregas** (`/entregas`): todas as entregas, com filtros por empresa, status e
  previsão.
- **Consultas de Pedidos** (`/pedidos/consultas`), com três abas:
  - **Pendências:** o que falta entregar ou cobrir;
  - **Prazos:** pedidos vencidos ou a vencer;
  - **Cancelamentos:** histórico, com o valor a devolver.

---

## 2.7 Site & Catálogo (estoque)

### Categorias (`/categorias`) — só administrador

A árvore tem dois níveis: raiz e subcategoria. O produto sempre fica numa
**subcategoria**.

### Produtos (`/produtos`) — todos veem; só administrador edita

- **Cartões:** total de produtos, ativos, valor em estoque e estoque baixo.
- **Filtros:** categoria, situação, destaque e "somente estoque baixo".
- **Novo Produto:**
  - Categoria, Nome, Código e Unidade (un, m, m², m³, kg, t, cx, sc, pct, lt);
  - Preço: vazio aparece como "Sob consulta" no site;
  - Preço promocional: menor que o preço cheio;
  - Estoque mínimo, Peso e Destaque.
- **Detalhe**, em painéis:
  - **Estoque por local:** saldo, reservado e disponível;
  - **Imagens:** PNG, JPEG ou WEBP;
  - **Reservas:** nova, baixar e liberar;
  - **Lotes e custo:** visível para administrador e financeiro;
  - **Movimentações.**

**Como dar entrada de 50 sacos de cimento:**
1. Abra o produto e clique em **Movimentar**.
2. Tipo **Entrada**, quantidade **50**, Local "Galpão Cabedelo".
3. Custo unitário "32,90" (obrigatório na entrada) e documento "NF 1234".
4. Clique em **Registrar**.

**Tipos de movimento:**

| Tipo | Efeito | Custo |
|---|---|---|
| Entrada | Soma ao saldo | Obrigatório |
| Saída / Perda | Subtrai do saldo, consumindo os lotes mais antigos primeiro (FIFO) | Proibido |
| Ajuste | Soma ou subtrai; é o único que aceita quantidade negativa | Opcional na entrada |
| Devolução | Soma ao saldo | Opcional |
| Implantação | Distribui num local o saldo antigo que ainda não estava alocado | — |

O saldo nunca fica negativo. Um lançamento errado se corrige com um movimento inverso.

**Como transferir entre locais:** clique em **Transferir** e informe Origem, Destino e
Quantidade. Os dois locais precisam ser da mesma empresa, e o custo do lote acompanha a
transferência.

**Como reservar sem pedido:** no painel Reservas, clique em **Nova reserva** e informe o
local e a quantidade. Depois use **Baixar** (gera a saída) ou **Liberar** (devolve ao
disponível). Reservas criadas por pedido só o próprio pedido mexe.

### Locais de Estoque (`/locais-estoque`)

- **Novo Local** (administrador): Empresa e Nome. A empresa não muda depois.
- **Ver saldos:** abre os produtos do local com saldo, reservado e disponível.
- Local com saldo não pode ser desativado. O caminho normal para tirar um local de uso é
  desativá-lo, não excluí-lo.

> **Vitrine pública (site):** mostra só produto ativo, com disponível maior que zero e de
> categoria ativa. Não exibe saldo nem custo.

---

## 2.8 Financeiro

### Fornecedores (`/fornecedores`) — todos veem; financeiro edita; administrador exclui

- **Fornecedor brasileiro:** País **BR**, CPF ou CNPJ (obrigatório), UF e cidade.
- **Fornecedor estrangeiro:** País (ex.: **CN**), identificação estrangeira e moeda
  (USD, EUR ou CNY).
- País e documento não mudam depois de salvos.

**Como cadastrar um fornecedor chinês:**
1. Clique em **Novo Fornecedor**.
2. País "CN", identificação estrangeira, nome "Foshan Ceramics", moeda **USD**.
3. Salve.

### Contas Bancárias (`/contas-bancarias`) — financeiro

- **Nova Conta:** Empresa, Banco, Tipo (Corrente, Poupança, Pagamento ou Aplicação),
  Agência, Conta e **Apelido** (ex.: "Sicredi movimento").
- Depois de criada, **só o apelido muda**. Para corrigir outro dado, desative a conta e
  cadastre outra.

### Plano de Contas (`/plano-de-contas`) — todos veem; financeiro edita

A árvore tem dois níveis.

1. **Raiz:** define a **Natureza** (Receita, Despesa ou Transferência) e a **Linha da DRE**
   (Receita operacional, Despesa fixa, Despesa variável, Resultado financeiro etc.).
2. **Subcategoria:** herda a natureza e a linha da raiz. É a única que fica **"Lançável"**,
   isto é, a única que pode ser usada em contas.

**Exemplo:**
1. Raiz "4.2 Despesas administrativas": Despesa, linha "Despesa fixa".
2. Subcategoria "4.2.01 Aluguel".

### Aprovadores (`/aprovadores`) — só administrador

Escolha o usuário e clique em **Designar**. Remover a designação mantém o histórico das
aprovações.

### Solicitações = Contas a pagar (`/solicitacoes`)

Não existe uma tela separada de "contas a pagar": **a solicitação é a conta a pagar**.

**Abas:**

| Aba | Quem vê | O que aparece |
|---|---|---|
| Minhas | Todos | As que a pessoa criou |
| Aprovação | Aprovadores | As pendentes de aprovação |
| Financeiro | Financeiro | Aprovadas, pagas, rejeitadas e canceladas |
| Todas | Administrador | Todas |

**Status:** `Pendente de aprovação → Aprovado → Pago`, ou `Rejeitado` ou `Cancelado`. O
estorno do pagamento volta a solicitação para Aprovado.

**Como pedir um pagamento (qualquer usuário):**
1. Clique em **Nova Solicitação**.
2. Preencha:
   - Empresa;
   - Valor ("1.500,00"), marcando "Valor estimado" se ainda não for o valor final;
   - Prazo (vencimento);
   - Forma (PIX, Boleto etc.);
   - Fornecedor;
   - Observação (obrigatória).
3. Anexe a nota ou o boleto (PDF ou imagem, até 10 MB).
4. Salve.

**Como aprovar (aprovador):**
1. Abra a aba **Aprovação** e clique no olho da linha.
2. Clique em **Aprovar**. Para recusar, clique em **Rejeitar** e escreva o motivo.

**Como pagar (financeiro):**
1. Abra a aba **Financeiro**, filtre o status "Aprovado" e abra a solicitação.
2. Se estiver sem categoria, clique em **Classificar** e informe a categoria e a
   competência.
3. Clique em **Registrar pagamento** e preencha a conta de onde saiu, o valor
   efetivamente pago, a data e o comprovante em PDF.

**Outras ações:**
- **Adiar:** novo vencimento, com motivo.
- **Estornar pagamento:** lançou errado.
- **Editar:** só enquanto estiver pendente, pelo solicitante.
- **Cancelar:** pelo solicitante ou administrador.

### Despesas Recorrentes (`/despesas-recorrentes`) — financeiro

Modelo de conta fixa mensal (aluguel, internet, contador).

**Como lançar o aluguel todo mês:**
1. Clique em **Nova Despesa** e preencha:
   - Empresa;
   - Descrição "Aluguel do galpão";
   - Valor;
   - Dia do vencimento 10;
   - Categoria "Aluguel";
   - Início 2026-01.
2. Todo mês, clique em **Gerar competência** (ex.: 2026-11). O sistema cria uma
   solicitação pendente de aprovação para cada despesa ativa. Rodar de novo não
   duplica.

### Resumo a Pagar (`/contas-a-pagar/resumo`) — financeiro

Tela só de leitura. Mostra os cartões Previsto, A pagar, Vencido e Estimado, e uma
tabela por semana e empresa.

---

## 2.9 Contas a receber (financeiro)

**Status do título:** `Em carteira`, `Em banco`, `Antecipado`, `Recebido` ou `Cancelado`.

**Origens:**
- **Manual:** digitado na tela.
- **Entrada de pedido:** gerado pelo pedido de importação.
- **Duplicata de NF:** gerado sozinho na importação do XML.

### Contas a Receber (`/titulos-receber`)

**Novo título manual:** Empresa, Cliente, Número, Parcela, Vencimento, Valor e Categoria
de receita. A Conta de cobrança é opcional: com ela o título já nasce "Em banco".

**Ações no detalhe:**

| Ação | Quando aparece | Efeito |
|---|---|---|
| Enviar ao banco | Em carteira | Passa a Em banco, na conta de cobrança escolhida |
| Registrar boleto enviado | Em banco | Grava o canal (e-mail, WhatsApp…) e a data |
| Retirar do banco | Em banco | Volta para carteira; lembre de baixar o boleto no banco |
| Prorrogar | Em aberto | Novo vencimento, com motivo |
| Registrar recebimento | Com saldo | Informe valor, juros e desconto; quando o saldo zera, vira Recebido |
| Estornar recebimento | Recebimento vigente | O saldo volta |
| Recomprar | Antecipado sem recebimento | O título volta para carteira e nasce uma conta a pagar ao fomento |
| Cancelar | Sem recebimento | Definitivo |

**Como dar baixa num boleto pago:**
1. Abra o título e clique em **Registrar recebimento**.
2. Informe a conta, a data e o valor recebido. Juros vão no campo próprio: não abatem o
   saldo.
3. Salve.

> **Caso especial:** se o título estava **antecipado** e o cliente pagou na conta da
> empresa (e não ao fomento), o sistema pede o vencimento e a categoria do **repasse** e
> gera sozinho a conta a pagar ao fomento.

### Antecipações (`/antecipacoes`)

**Como antecipar títulos com um fomento:**
1. Clique em **Nova antecipação**.
2. Preencha:
   - Empresa;
   - Fomento (cadastrado como fornecedor);
   - Nº do borderô;
   - Data;
   - Conta do crédito.
3. Marque os títulos. A tela mostra o **Bruto**, que é a soma dos saldos.
4. Informe o **Valor líquido** que efetivamente caiu na conta.
5. Salve. Os encargos (deságio) = bruto − líquido. A taxa é só informativa.

Se o borderô foi lançado errado, use **Estornar borderô**. Só é possível se nenhum título
dele foi recebido nem recomprado.

### Resumo a Receber e Boletos Pendentes

- **Resumo a Receber:** cartões A receber, Antecipado e Vencido, e uma tabela por semana.
- **Boletos Pendentes:** títulos em banco que ainda não tiveram o boleto enviado ao
  cliente. Registre o envio direto da lista.

---

## 2.10 Notas Fiscais (`/notas-fiscais`)

O sistema **não emite** nota: ele **importa o XML autorizado**.

**Como importar uma NF-e (financeiro):**
1. Clique em **Importar NF-e** e escolha o `.xml` do `nfeProc`.
2. O sistema identifica a empresa emitente, o cliente e os produtos.
   - Cliente que não existe é criado automaticamente.
   - Nota de **venda** com duplicatas gera os **títulos a receber** sozinha.
3. Leia os avisos: itens sem produto, cliente criado, "o cliente exige nº do pedido na NF
   e o XML não traz", títulos pendentes.
4. Abra a nota e:
   - **Vínculo:** ligue a nota ao pedido e à entrega;
   - ligue os itens sem produto;
   - se houve aviso de títulos pendentes, clique em **Sincronizar títulos**.

> A NF **não mexe no estoque**. A baixa acontece na **saída da entrega** do pedido.

**Como cancelar uma nota:** clique em **Importar evento** e envie o XML do
`procEventoNFe` de cancelamento. O sistema cancela a nota, libera a entrega, cancela os
títulos sem movimento e marca os que já foram recebidos ou antecipados.

**Telas de pendência:**
- **Itens sem Produto:** liga os itens das notas ao cadastro de produtos.
- **Entregas sem NF:** entregas que saíram e ainda não têm nota.

---

# Parte 3 — Backend (API por módulo)

Repo: `D:\PROJETOS\GOLANG\painelMiaConstrutora`. Todas as rotas ficam em
`internal/adapter/http/rotas.go`; o contrato completo está em `API.md`.

## 3.1 Mecânica comum

**Base e autenticação**
- Base: `/api/v1`. O token vai no cabeçalho `Authorization: Bearer <token>`.
- Rotas públicas: `/health`, `/health/db`, `POST /usuarios/autenticar` e `/publico/*`
  (vitrine).

**Proteção das rotas**
- `protegido`: qualquer usuário logado.
- `comercial`, `financeiro` e `administrador`: perfil exigido. O administrador passa em
  todos.
- As regras finas ficam em `usecases/<modulo>/permissoes.go`.

**Formato das respostas**
- Sucesso: `{"dados": ...}`.
- Lista: `{"dados":[...],"pagina":1,"tamanho":20}`.
- Erro: `{"erro":{"codigo":409,"mensagem":"..."}}`.

**Paginação:** `?pagina=` (padrão 1) e `?tamanho=` (padrão 20, máximo 100).

**Corpo JSON**
- Limite de 1 MB.
- Campo desconhecido dá **400**.
- Id que não é UUID dá **400**.

**Upload de arquivos**
- O upload é sempre feito primeiro, e depois o arquivo é referenciado pelo `arquivo_id`.
- Envio: `POST /arquivos` em multipart, com o campo `arquivo`, até 10 MB.
- Tipos aceitos: pdf, png, jpeg, webp e xml.

**Sessão**
- O JWT dura 480 minutos.
- Trocar senha ou chamar `POST /sessao/encerrar` avança `versao_sessao` e derruba todos
  os tokens anteriores.

**Concorrência otimista:** as operações de mudança mandam `{"versao": n}`. Versão
divergente dá **409**.

**Exemplo — login e uso do token:**
```
POST /api/v1/usuarios/autenticar
{"email":"admin@exemplo.com","senha":"..."}
→ {"dados":{"token":"eyJ...","expira_em":"...","administrador":true,"comercial":true,"financeiro":true,"usuario":{...}}}

GET /api/v1/sessao
Authorization: Bearer eyJ...
```

## 3.2 Usuários, cargos, sessão e aprovadores

| Rota | Perfil |
|---|---|
| `POST/GET /usuarios`, `GET/PUT/DELETE /usuarios/{id}`, `POST /usuarios/{id}/senha`, `PATCH /usuarios/{id}/ativar` e `/desativar` | administrador |
| `GET/PUT /usuarios/{id}/empresas` | administrador |
| `GET /sessao`, `POST /sessao/senha`, `POST /sessao/encerrar` | protegido |
| `GET /cargos`, `/cargos/{id}`, `/cargos/busca?nome=` | protegido |
| `POST/PUT/DELETE /cargos` | administrador |
| `POST/GET /aprovadores`, `DELETE /aprovadores/{id}` | administrador |

**Regras**
- Login errado dá **404** "email ou senha inválidos", com tempo de resposta constante.
- Força bruta: 5 falhas do mesmo e-mail no mesmo IP, ou 50 falhas do mesmo IP, em 15
  minutos, dão **429**.
- Qualquer operação que deixe o sistema sem administrador ativo dá **409**.

**Exemplo:**
```
POST /api/v1/cargos    {"nome":"Comprador","administrador":false,"financeiro":false,"comercial":false}
POST /api/v1/usuarios  {"nome":"Ana Souza","email":"ana@mia.com.br","senha":"senhaForte123","cargo_id":"<cargo>"}
PUT  /api/v1/usuarios/<ana>/empresas  {"acesso_todas":false,"empresa_ids":["<mia>"]}
POST /api/v1/aprovadores {"usuario_id":"<ana>"}
```

## 3.3 Leads e funis

- **Comercial:** CRUD de `/leads`, `PATCH /leads/{id}/etapa`, `GET /leads/{id}/historico`,
  as contagens `/funils/{id}/leads/contagem` e `/etapas/{id}/leads/contagem`, e
  `POST /leads/{id}/converter`.
- **Administrador:** escrita de `/funils` e `/etapas`, e `PUT /funils/{id}/etapas/ordem`
  (precisa trazer **todas** as etapas do funil).

**Regras**
- O `PUT` do lead não muda a etapa. Para isso existe o `PATCH /etapa`, que valida a etapa
  de destino (mesmo funil e ativa) e grava o histórico na mesma transação.
- Lead já convertido em cliente não pode ser excluído.

**Exemplo:**
```
POST  /api/v1/leads {"nome":"Construtora Águia","email":"compras@aguia.com.br","telefone":"83 99999-0000","origem":"site","etapa_id":"<novo contato>"}
PATCH /api/v1/leads/<id>/etapa {"etapa_id":"<proposta>"}
```

## 3.4 M1 — Empresas

**Endpoints**
- **Administrador:** `POST /empresas`, `PUT /empresas/{id}` (mandar `cnpj` no corpo dá
  **400**), `PATCH /empresas/{id}/ativar` e `/desativar`, `DELETE /empresas/{id}`.
- **Protegido:** `GET /empresas`, já filtrado pelo acesso do usuário.

**Regras**
- O CNPJ é validado pelos dígitos verificadores e aceita o formato alfanumérico.
- CNPJ duplicado dá **409**.
- Usuário novo nasce **sem empresa**.
- Fora do acesso do usuário, a resposta é **403**, inclusive para id inexistente.

```
POST /api/v1/empresas {"cnpj":"11222333000181","razao_social":"MIA Construtora LTDA","nome_fantasia":"MIA","uf":"PB","regime_tributario":"presumido"}
```

## 3.5 M4 — Fornecedores

**Endpoints**
- **Financeiro:** `POST /fornecedores`, `PUT /fornecedores/{id}` (com `versao`; substitui
  o cadastro inteiro), `PATCH /fornecedores/{id}/ativar` e `/desativar` com `{versao}`.
- **Protegido:** `GET /fornecedores` (filtros `busca`, `pais`, `moeda`, `ativo`).
- **Administrador:** `DELETE /fornecedores/{id}`.

**Regras**
- País BR: o documento é obrigatório.
- Outros países: o documento é proibido; usa-se `identificacao_estrangeira`, única por
  país.

```
POST /api/v1/fornecedores {"documento":"11222333000181","nome":"Cerâmica Paraíba LTDA","moeda_padrao":"BRL","uf":"PB","cidade":"João Pessoa"}
POST /api/v1/fornecedores {"pais":"CN","identificacao_estrangeira":"91330200MA2XXXXX","nome":"Foshan Ceramics","moeda_padrao":"USD"}
```

## 3.6 M5 — Contas bancárias e plano de contas

**Contas bancárias**
- **Financeiro:** CRUD de `/contas-bancarias`. O `PUT` aceita só `{apelido, versao}`.
- **Protegido:** `GET /bancos` (catálogo COMPE).
- Regras:
  - mesma conta duas vezes no grupo dá **409**;
  - apelido repetido na mesma empresa dá **409**;
  - a conta não guarda saldo (isso fica para o M13).

**Plano de contas**
- **Financeiro:** escrita de `/categorias-financeiras`.
- **Protegido:** `GET /categorias-financeiras` e `GET /categorias-financeiras/arvore`.
- Regras:
  - raiz exige `natureza` + `linha_dre`, e a combinação é validada;
  - subcategoria não aceita esses campos: herda da raiz;
  - só a subcategoria ativa é `lancavel`.

```
POST /api/v1/contas-bancarias {"empresa_id":"<mia>","banco":"748","tipo":"corrente","agencia":"0812","conta":"54321-0","apelido":"Sicredi movimento"}
POST /api/v1/categorias-financeiras {"nome":"Despesas administrativas","natureza":"despesa","linha_dre":"despesa_fixa","codigo":"4.2","ordem":3}
POST /api/v1/categorias-financeiras {"nome":"Aluguel","categoria_pai_id":"<4.2>","codigo":"4.2.01"}
```

## 3.7 Estoque, M2 (locais, lotes e reservas) e vitrine

**Rotas do administrador**
- Escrita de `/categorias` e `/produtos`.
- `PATCH /produtos/{id}/destaque`.
- `POST /produtos/{id}/movimentos`, `/transferencias`, `/reservas` e `/imagens`.
- `POST /reservas/{id}/liberar`.
- CRUD de `/estoque/locais`.

**Rotas de leitura (protegido)**
- `GET /produtos` (filtros `categoria_id`, `busca`, `ativo`, `destaque`, `estoque_baixo`).
- `GET /produtos/{id}/saldo`, `/movimentos` e `/reservas`.
- `GET /estoque/resumo`.
- `GET /estoque/locais/{id}/saldos`.
- `GET /produtos/{id}/lotes`: só administrador ou financeiro, que é a regra de quem vê o
  custo.

**Rotas públicas (vitrine)**
- `GET /publico/categorias`, `/publico/produtos?categoria=<slug>`,
  `/publico/produtos/{slug}`, `/publico/destaques` e `/publico/imagens/{id}`.

**Regras**
- Custo por lote, consumido em **FIFO**. O saldo nunca fica negativo.
- O movimento `transferencia` só passa pela rota própria; na rota de movimentos dá
  **400**.
- Reserva:
  - sai do disponível;
  - é baixada por uma saída com o mesmo `reserva_id`, o mesmo local e a mesma quantidade;
  - reserva de origem `pedido_item` só é mexida pelo pedido (**409**).
- Local:
  - com saldo, não pode ser desativado;
  - inativo, não recebe entrada.

```
POST /api/v1/produtos {"categoria_id":"<sub>","nome":"Cimento CP II 50kg","codigo":"SKU-001","unidade_medida":"sc","preco_centavos":3990,"estoque_minimo":20,"peso_kg":50}
POST /api/v1/produtos/<id>/movimentos {"local_id":"<galpão>","tipo":"entrada","quantidade":200,"custo_unitario_centavos":3100,"documento_ref":"NF-100"}
POST /api/v1/produtos/<id>/transferencias {"local_origem_id":"<galpão>","local_destino_id":"<loja>","quantidade":40,"documento_ref":"ROM-12"}
POST /api/v1/produtos/<id>/reservas {"local_id":"<loja>","quantidade":10,"observacao":"pedido Águia"}
```

## 3.8 M3 — Clientes, obras, grupos e representantes

**Endpoints**
- **Comercial:**
  - escrita de `/clientes`, `/clientes/{id}/obras`, `/obras/{id}` e `/cliente-grupos`;
  - `POST /leads/{id}/converter`;
  - `GET /representantes`.
- **Administrador:**
  - `DELETE /clientes/{id}`;
  - escrita de `/representantes` e `PUT /representantes/{id}/fornecedor`;
  - `POST /clientes/importacao?simular=true` (CSV cru, separado por `;`).

**Regras**
- O documento do cliente (CPF ou CNPJ) é único e imutável.
- Tipo `pf` exige CPF, e CPF exige o tipo `pf`.
- Comissão do representante em pontos-base: `comissao_pronta_entrega_bp` e
  `comissao_importacao_bp`.
- Representante com login só vê a própria carteira; cliente de outro representante dá
  **403** "cliente de outro representante".

```
POST /api/v1/clientes {"documento":"11222333000181","nome":"Construtora Águia LTDA","tipo":"construtora","uf":"PB","cidade":"João Pessoa","representante_id":"<rep>","email_nf":"nf@aguia.com.br","exige_numero_pedido_na_nf":true,"proibe_antecipacao":false}
POST /api/v1/clientes/<id>/obras {"nome":"Residencial Mar Azul","logradouro":"Av. Beira Mar","numero":"100","cidade":"Cabedelo","uf":"PB","cep":"58310000"}
```

## 3.9 M6 — Pedidos e entregas

**Endpoints**
- **Comercial:**
  - `POST /pedidos` e `PUT /pedidos/{id}`;
  - `POST /pedidos/{id}/confirmar`, `/reservar-pendencias` e `/cancelar`;
  - `POST /pedidos/{id}/entregas`;
  - `POST /entregas/{id}/saiu`, `/entregue` e `/cancelar`.
- **Financeiro:** `POST /pedidos/{id}/entrada`.
- **Protegido:**
  - `GET /pedidos`, `/pedidos/{id}`, `/pedidos/{id}/historico` e `/pedidos/{id}/entregas`;
  - `GET /entregas`;
  - `GET /pedidos/pendencias`, `/pedidos/prazos` e `/pedidos/cancelamentos`.

**Regras**
- **Numeração:** o `numero` é sequencial por empresa.
- **Itens:** só mudam em `proposta`. Preço zero é aceito (brinde).
- **Confirmação:** reserva por item, começando pelo local com mais disponível. O que não
  houver em estoque fica "a cobrir" e não bloqueia a confirmação.
- **Saída da entrega** (`saiu`): baixa o estoque na mesma transação e consome a reserva.
  Sem saldo no local dá **400**.
- **Entrada:** só no modelo importação, uma vez só, com data não futura. Grava
  `prazo_limite` = data da entrada + 120 dias.
- **Cancelamento:**
  - motivo obrigatório;
  - libera as reservas;
  - entrega em trânsito dá **409**;
  - `valor_a_devolver_centavos` fica só registrado.

**Fluxo completo:**
```
POST /api/v1/pedidos {"empresa_id":"…","cliente_id":"…","obra_id":"…","modelo":"importacao","numero_pedido_cliente":"OC-778","validade":"2026-11-03","itens":[{"produto_id":"…","quantidade":10,"preco_unitario_centavos":1500}]}
POST /api/v1/pedidos/<id>/confirmar        {"versao":1}
POST /api/v1/pedidos/<id>/titulo-entrada   {"vencimento":"2026-10-25"}
POST /api/v1/pedidos/<id>/entregas         {"frete_tipo":"cif","frete_centavos":18000,"itens":[{"pedido_item_id":"…","local_id":"…","quantidade":5}]}
POST /api/v1/entregas/<id>/saiu            {"versao":1,"saiu_em":"2026-10-19"}
POST /api/v1/entregas/<id>/entregue        {"versao":2,"entregue_em":"2026-10-20"}
```

## 3.10 M7 — Contas a pagar (solicitações e despesas recorrentes)

**Status:** `pendente_aprovacao → aprovado | rejeitado | cancelado`;
`aprovado → pago | cancelado`; `pago → aprovado` (só por estorno).

| Ação | Rota | Quem pode |
|---|---|---|
| Criar | `POST /solicitacoes` | Qualquer usuário, com a empresa no acesso |
| Listar | `GET /solicitacoes?escopo=minhas\|aprovacao\|financeiro\|todas` | Depende do escopo |
| Aprovar / Rejeitar | `POST /solicitacoes/{id}/aprovar` e `/rejeitar` | Aprovador ou administrador, **nunca o próprio solicitante** |
| Editar / Cancelar | `PUT /solicitacoes/{id}` e `POST /solicitacoes/{id}/cancelar` | Solicitante ou administrador |
| Classificar | `PATCH /solicitacoes/{id}/classificacao` | Financeiro, com status aprovado |
| Adiar | `POST /solicitacoes/{id}/adiar` | Financeiro |
| Pagar / Estornar | `POST /solicitacoes/{id}/pagamento` e `/pagamento/estorno` | Financeiro |
| Contadores | `GET /solicitacoes/pendencias` | Protegido |
| Recorrentes | `/despesas-recorrentes` (CRUD) e `POST /despesas-recorrentes/gerar?competencia=AAAA-MM` | Financeiro |
| Resumo | `GET /contas-a-pagar/resumo` | Financeiro |

**Regras do pagamento**
- A solicitação precisa estar classificada, com subcategoria de despesa.
- A conta bancária precisa estar ativa e ser da mesma empresa.
- O comprovante precisa ser um PDF enviado pelo próprio usuário que registra.
- `pago_em` não pode estar no futuro.

**Contas geradas por outros módulos**
- Origens: recompra, repasse ao fomento e comissão (no futuro também importação e
  dívida).
- Todas entram como `pendente_aprovacao`, já classificadas, sem duplicar
  (`uq_solicitacao_origem`).

**Fluxo completo:**
```
POST  /api/v1/arquivos                         (PDF da nota) → arquivo_id
POST  /api/v1/solicitacoes {"empresa_id":"…","valor_centavos":150000,"prazo_pagamento":"2026-10-30","observacao":"Cimento fase 2","forma_pagamento":"pix","fornecedor_id":"…","competencia":"2026-10","arquivo_ids":["…"]}
POST  /api/v1/solicitacoes/<id>/aprovar        {"versao":1}
PATCH /api/v1/solicitacoes/<id>/classificacao  {"fornecedor_id":"…","categoria_financeira_id":"<sub>","competencia":"2026-10","versao":2}
POST  /api/v1/solicitacoes/<id>/pagamento      {"conta_bancaria_id":"…","valor_centavos":148500,"comprovante_arquivo_id":"…","pago_em":"2026-10-28T14:00:00Z"}
POST  /api/v1/solicitacoes/<id>/pagamento/estorno {"motivo":"duplicidade","versao":4}
```

## 3.11 M8 — NF-e por XML

**Endpoints**
- **Financeiro:**
  - `POST /notas-fiscais/importacao`;
  - `PUT /notas-fiscais/{id}/vinculo`;
  - `PUT /notas-fiscais/{id}/itens/{numero}/produto`;
  - `POST /notas-fiscais/eventos`;
  - `POST /notas-fiscais/{id}/titulos`.
- **Protegido:**
  - `GET /notas-fiscais`, `/notas-fiscais/{id}`, `/notas-fiscais/{id}/xml` e
    `/notas-fiscais/{id}/eventos`;
  - `GET /notas-fiscais/itens-sem-produto`;
  - `GET /entregas/sem-nota-fiscal`.

**XML recusado com 400**
- Sem protocolo de autorização.
- Modelo 65 ou ambiente de homologação.
- `cStat` diferente de 100 ou 150.
- Chave inválida.
- Emitente que não está cadastrado como empresa do grupo.

**Outras regras**
- Chave repetida dá **409**.
- Destinatário do grupo vira nota `entre_empresas` (transferência) e não gera título.
  Qualquer outro destinatário vira cliente, criado se não existir.
- Nota de venda autorizada com duplicatas gera um título a receber por duplicata, com
  origem `nota_fiscal_duplicata`.
- O evento de cancelamento cancela a nota, libera a entrega e cancela ou marca os
  títulos.
- A nota não move estoque.

```
POST /api/v1/arquivos  (XML)                      → arquivo_id
POST /api/v1/notas-fiscais/importacao {"arquivo_id":"…","pedido_id":"…","entrega_id":"…"}
→ 201 com duplicatas, avisos e titulos_gerados
POST /api/v1/notas-fiscais/eventos {"arquivo_id":"<xml do procEventoNFe>"}
```

## 3.12 M9 — Contas a receber e antecipação

Todas as rotas exigem o perfil financeiro.

**Endpoints**
- **Títulos:**
  - `POST/GET /titulos-receber` e `GET/PUT /titulos-receber/{id}`;
  - `POST /titulos-receber/{id}/prorrogar`, `/enviar-banco`, `/retirar-banco`,
    `/boleto-enviado` e `/cancelar`;
  - `GET /titulos-receber/{id}/historico`.
- **Recebimentos:**
  - `POST/GET /titulos-receber/{id}/recebimentos`;
  - `POST /titulos-receber/{id}/recebimentos/{rid}/estorno`;
  - `POST /titulos-receber/{id}/recompra`.
- **Antecipações:** `POST/GET /antecipacoes`, `GET /antecipacoes/{id}` e
  `POST /antecipacoes/{id}/estorno`.
- **Outros:** `POST /pedidos/{id}/titulo-entrada`, `GET /titulos-receber/boletos-pendentes`
  e `GET /contas-a-receber/resumo`.
- Não existe `DELETE`: título errado é cancelado.

**Cálculos**
- **Recebimento:**
  - `valor + desconto <= saldo`;
  - os juros não abatem o saldo;
  - o título vira `recebido` quando o saldo zera.
- **Antecipação:**
  - bruto = soma dos saldos dos títulos;
  - o líquido é digitado;
  - encargos = bruto − líquido;
  - a taxa é só informativa;
  - cliente com `proibe_antecipacao` bloqueia o borderô inteiro.
- **Recompra:** gera a conta a pagar ao fomento com saldo + encargos.
- **Título de entrada do pedido:**
  - valor = `total × entrada_bp / 10000`;
  - pedir de novo devolve o mesmo título;
  - a baixa que zera o saldo grava a entrada no pedido.

```
POST /api/v1/titulos-receber/<id>/enviar-banco {"versao":1,"conta_bancaria_id":"…"}
POST /api/v1/antecipacoes {"empresa_id":"…","fornecedor_id":"<fomento>","numero":"B-77","operado_em":"2026-10-15","taxa_bp":300,"valor_liquido_centavos":294000,"conta_bancaria_id":"…","titulo_ids":["…","…"]}
POST /api/v1/titulos-receber/<id>/recebimentos {"versao":3,"tipo":"fomento","recebido_em":"2026-11-09","valor_centavos":150000}
POST /api/v1/titulos-receber/<id>/recompra {"versao":3,"recomprado_em":"2026-11-12","encargos_centavos":4500,"vencimento":"2026-11-20","categoria_financeira_id":"…","motivo":"cliente não pagou"}
```

## 3.13 M10 — Comissões (API pronta; front em andamento)

**Como a comissão é calculada**
- Base = Σ (quantidade que saiu × preço unitário). Não entra frete nem o valor da NF.
- Valor = `(base × percentual_bp + 5000) / 10000`, arredondado meio para cima.
- Percentual:
  - o padrão vem do representante, conforme o modelo do pedido;
  - pode ser trocado por pedido antes da primeira saída;
  - fica congelado na primeira apuração.
- Gera uma comissão por entrega que saiu, com competência no mês da saída.

**Fechamento**
- Só para mês já encerrado.
- Agrupa por empresa, representante e competência.
- Gera a conta a pagar no M7 com vencimento no dia 15 do mês seguinte, em nome do
  fornecedor ligado ao representante.
- Não é desfeito: correção vira ajuste no mês seguinte.

**Permissões**
- `PUT /pedidos/{id}/comissao`: comercial (nunca o próprio representante).
- Apurar, ajustar e fechar: financeiro que não seja representante.
- Leitura: financeiro, ou o representante (só as próprias).

```
PUT  /api/v1/pedidos/<id>/comissao {"percentual_bp":450,"versao":1}
POST /api/v1/comissoes/apurar {"empresa_id":"…"}
POST /api/v1/comissoes/ajustes {"empresa_id":"…","representante_id":"…","competencia":"2026-09","valor_centavos":-375,"motivo":"devolução de uma peça"}
POST /api/v1/comissoes/fechamentos {"empresa_id":"…","competencia":"2026-09","categoria_financeira_id":"…"}
```

## 3.14 M11 — Propostas e tabela de preços (API pronta; sem front)

**Tabela de preços**
- Só recebe acréscimos: mudar preço é lançar uma nova vigência (`vigente_de`).
- Preço vigente = a maior vigência até a data. Sem tabela, usa o preço do produto.
- A tabela só **sugere** preço, não trava o pedido.

**Proposta**
- É o próprio pedido em `proposta`, com `validade` (padrão hoje + 15 dias).
- Cada `PUT` aumenta a `revisao`.
- Confirmar uma proposta vencida dá **409**.
- `GET /pedidos/{id}/proposta` devolve o documento pronto para impressão
  ("Proposta 57-R2").

**Permissões**
- `POST /tabela-precos`: administrador.
- `DELETE /tabela-precos/{id}`: administrador, e só para vigência futura.
- Leitura: comercial, financeiro ou administrador.

```
POST /api/v1/tabela-precos {"produto_id":"…","modelo":"pronta_entrega","preco_centavos":12000,"vigente_de":"2026-11-01","observacao":"reajuste"}
GET  /api/v1/tabela-precos/vigentes?modelo=pronta_entrega
GET  /api/v1/pedidos/<id>/proposta
```

## 3.15 Módulos pendentes

| Módulo | O que vai fazer |
|---|---|
| M12 Importações | Processo de importação (embarque até recebimento), câmbio e custos rateados nos lotes; pagamentos ao fornecedor viram contas a pagar; o recebimento reserva automaticamente as pendências dos pedidos |
| M13 Caixa e conciliação | Extrato OFX/CSV, conciliação com pagamentos e recebimentos, lançamentos avulsos, fechamento mensal e projeção de 90 dias. Escopo em `CAIXA.md`, com decisões em aberto |
| M14 Dívidas e empréstimos | Dívidas e parcelas, que viram contas a pagar; renegociação |
| M15 DRE e painéis | DRE por competência e linha da DRE, faturamento e relatório semanal (só leitura) |

---

# Parte 4 — Lacunas e inconsistências encontradas

**Front x backend**
1. **Comissões:** nenhuma tela ainda (F-M10 em andamento). Falta também ligar o
   representante a um fornecedor (`PUT /representantes/{id}/fornecedor`), sem o qual o
   fechamento falha.
2. **Propostas e tabela de preços:**
   - não há tela de tabela de preços;
   - não há botão para imprimir a proposta;
   - o front usa uma lista fixa de modelos em vez de `GET /modelos-venda`;
   - o preço sugerido vem só do cadastro do produto.
3. **Páginas provisórias:** Negociações, Formulários, Relatórios e Configurações.

**Regras de negócio**
4. **Fuso no M7:** o "hoje" do M7 é calculado em UTC, enquanto M6, M9, M10 e M11 usam o
   horário de Brasília (pendência 11 do `PENDENCIAS.md`). Perto da meia-noite, a
   validação de "data futura" pode divergir.
5. **Cancelamento de pedido:**
   - não cancela o título de entrada: o financeiro cancela à mão (decisão D3 do M9);
   - o valor a devolver fica só registrado, sem gerar conta a pagar.

**Permissões e navegação**
6. **Permissão fora da rota:** `GET /tabela-precos*` e as leituras de `/comissoes*` são
   `protegido` na rota, mas o usecase restringe o perfil. O resultado é correto, mas a
   rota não documenta a restrição.
7. **Rota proibida no front:** redireciona para o Dashboard sem nenhuma mensagem.
