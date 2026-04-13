# JUnit & Mockito Conventions

## Table of Contents
1. [Naming Conventions](#naming-conventions)
2. [Estructura de Archivos](#estructura-de-archivos)
3. [Antipatrones](#antipatrones)
4. [Test Coverage Strategy](#test-coverage-strategy)
5. [Spring Integration Tests](#spring-integration-tests)

---

## Naming Conventions

### Clases de test
```
{ClaseBajoTest}Test.java
```
Ejemplos: `TransferServiceTest.java`, `TransferValidatorTest.java`, `TransferMapperTest.java`

### Metodos de test
```
should{ComportamientoEsperado}_When{Condicion}
```
Ejemplos:
- `shouldCompleteTransfer_WhenAccountsValidAndSufficientFunds`
- `shouldThrowInsufficientFunds_WhenBalanceBelowAmount`
- `shouldReturnEmptyList_WhenNoTransfersFound`
- `shouldMapAllFields_WhenEntityIsComplete`

### Object Mothers
```
{Entidad}Mother.java
```
Metodos: `createDefault()`, `createValid*()`, `createWith*()`, `createInvalid*()`

### Constantes de test
```java
private static final String SOURCE_ACCOUNT_ID = "ACC-001";
private static final String TARGET_ACCOUNT_ID = "ACC-002";
private static final BigDecimal TRANSFER_AMOUNT = new BigDecimal("1000.00");
```

---

## Estructura de Archivos

```
src/test/java/com/example/
├── service/
│   └── TransferServiceTest.java
├── validator/
│   └── TransferValidatorTest.java
├── mapper/
│   └── TransferMapperTest.java
├── controller/
│   └── TransferControllerTest.java   (con @WebMvcTest)
└── mother/
    ├── TransferRequestMother.java
    ├── TransferResponseMother.java
    ├── TransferMother.java
    └── AccountMother.java
```

**Reglas:**
- Estructura de paquetes espejo de `src/main/java`
- Mothers en paquete separado `mother` (compartidos entre tests)
- Un archivo de test por clase de produccion

---

## Antipatrones

### Prohibido
```java
// NO: @SpringBootTest para tests unitarios (lento, innecesario)
@SpringBootTest
class TransferServiceTest { ... }

// NO: when/verify de Mockito (usar BDDMockito)
when(repo.findById(any())).thenReturn(Optional.of(account));
verify(repo).save(any());

// NO: assertTrue/assertFalse con condiciones (poco descriptivo)
assertTrue(result.isPresent());
assertFalse(list.isEmpty());

// NO: Datos hardcodeados repetidos en multiples tests
@Test void test1() { var request = new TransferRequest("ACC-001", "ACC-002", ...); }
@Test void test2() { var request = new TransferRequest("ACC-001", "ACC-002", ...); }

// NO: Test sin @DisplayName
@Test
void test() { ... }

// NO: Multiples acciones en When
final var intermediate = service.prepare(request);
final var result = service.execute(intermediate);

// NO: Catch manual de excepciones
@Test
void shouldThrow() {
    try {
        service.execute(request);
        fail("Should have thrown");
    } catch (Exception e) {
        assertEquals("message", e.getMessage());
    }
}

// NO: @Test(expected = ...) — no permite verificar mensaje
@Test(expected = CustomException.class)
void shouldThrow() { ... }

// NO: Mock de todo — solo mockear dependencias externas
@Mock private BigDecimal amount; // NO mockear value objects
```

### Correcto
```java
// SI: @ExtendWith para tests unitarios
@ExtendWith(MockitoExtension.class)
class TransferServiceTest { ... }

// SI: BDDMockito
given(repo.findById(any())).willReturn(Optional.of(account));
then(repo).should().save(any());

// SI: AssertJ descriptivo
assertThat(result).isPresent();
assertThat(list).isNotEmpty();

// SI: Object Mothers
final var request = TransferRequestMother.createValidRequest();

// SI: @DisplayName descriptivo
@DisplayName("Should complete transfer when accounts are valid")

// SI: Una sola accion en When
final var result = service.execute(request);

// SI: assertThatThrownBy
assertThatThrownBy(() -> service.execute(request))
    .isInstanceOf(CustomException.class)
    .hasMessageContaining("message");
```

---

## Test Coverage Strategy

### Que testear por capa

| Capa | Tipo de test | Que verificar |
|------|-------------|---------------|
| Service | Unitario con mocks | Orquestacion correcta, delegacion a validator/mapper/repo |
| Validator | Unitario sin mocks | Cada regla de validacion, excepciones correctas |
| Mapper | Unitario sin mocks | Transformacion completa de campos |
| Controller | `@WebMvcTest` | Status codes, validacion Jakarta, serializacion JSON |
| Repository | `@DataJpaTest` | Queries custom, solo si hay `@Query` |

### Prioridad de tests
1. **Caso feliz** (happy path) — siempre primero
2. **Validaciones y errores de negocio** — cada regla del validator
3. **Casos limite** — null, empty, valores extremos
4. **Interacciones** — orden y frecuencia de llamadas cuando importa

### Que NO testear
- Getters/setters generados por Lombok
- Delegaciones triviales (metodo que solo llama a otro)
- Configuracion de Spring (beans, properties)
- Codigo generado por frameworks

---

## Spring Integration Tests

Solo para Controller y Repository. Nunca para Service/Validator/Mapper.

### Controller con @WebMvcTest
```java
@WebMvcTest(TransferController.class)
class TransferControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockitoBean
    private TransferService transferService;

    @Test
    @DisplayName("Should return 201 when transfer is created successfully")
    void shouldReturn201_WhenTransferCreated() throws Exception {
        // Given
        final var request = TransferRequestMother.createValidRequest();
        final var response = TransferResponseMother.createCompleted();
        given(transferService.execute(any(TransferRequest.class))).willReturn(response);

        // When / Then
        mockMvc.perform(post("/api/v1/transfers")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
            .andExpect(status().isCreated())
            .andExpect(jsonPath("$.id").value(response.id()))
            .andExpect(jsonPath("$.status").value("COMPLETED"));
    }

    @Test
    @DisplayName("Should return 400 when request is invalid")
    void shouldReturn400_WhenRequestInvalid() throws Exception {
        // Given
        final var invalidRequest = "{}";

        // When / Then
        mockMvc.perform(post("/api/v1/transfers")
                .contentType(MediaType.APPLICATION_JSON)
                .content(invalidRequest))
            .andExpect(status().isBadRequest());
    }
}
```

### Repository con @DataJpaTest
```java
@DataJpaTest
class TransferRepositoryTest {

    @Autowired
    private TransferRepository transferRepository;

    @Test
    @DisplayName("Should find transfers by status and date")
    void shouldFindTransfersByStatusAndDate() {
        // Given
        final var transfer = TransferMother.createPending();
        transferRepository.save(transfer);

        // When
        final var results = transferRepository.findRecentByStatus(
            TransferStatus.PENDING, Instant.now().minus(Duration.ofHours(1)));

        // Then
        assertThat(results).hasSize(1);
        assertThat(results.getFirst().getStatus()).isEqualTo(TransferStatus.PENDING);
    }
}
```

**Reglas:**
- `@WebMvcTest` carga solo el controller (rapido)
- `@MockitoBean` para mockear dependencias del controller (no `@MockBean` deprecado)
- `@DataJpaTest` carga solo JPA (rapido, con H2 embebido)
- Verificar status codes, JSON paths, validacion Jakarta
- NO usar `@SpringBootTest` salvo tests E2E que realmente lo necesiten
