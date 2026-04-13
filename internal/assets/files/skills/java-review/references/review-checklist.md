# Checklist de Code Review — Java / Spring Boot

Referencia detallada para revisiones de código en aplicaciones Java con Spring Boot. Aplica todas las categorías relevantes al código bajo revisión.

## Índice

- [1 - Style Guide y Convenciones](#1---style-guide-y-convenciones)
- [2 - Security Patterns](#2---security-patterns)
- [3 - Performance y Optimización](#3---performance-y-optimización)
- [4 - Test Quality](#4---test-quality)
- [5 - Spring Boot Best Practices](#5---spring-boot-best-practices)

---

## 1 - Style Guide y Convenciones

### Naming Conventions

| Elemento | Convención | Ejemplo |
|----------|-----------|---------|
| Clases, interfaces, enums, records | UpperCamelCase | `UserService`, `PaymentStatus` |
| Métodos, variables, parámetros | lowerCamelCase | `getUserById`, `totalAmount` |
| Constantes (static final) | UPPER_SNAKE_CASE | `MAX_RETRY_COUNT`, `DEFAULT_TIMEOUT` |
| Paquetes | lowercase, sin guiones | `com.example.userservice` |
| Type parameters | letra mayúscula sola | `T`, `E`, `K`, `V` |
| Interfaces | sin prefijo I | `UserRepository` (no `IUserRepository`) |
| Implementaciones | sufijo descriptivo | `UserRepositoryImpl`, `JpaUserRepository` |

### Estructura de Paquetes

Seguir estructura por capas o por features:

```
# Por capas (proyectos pequeños)
com.example.app/
├── controller/
├── service/
├── repository/
├── model/ (entities)
├── dto/
├── config/
└── exception/

# Por features (proyectos grandes)
com.example.app/
├── user/
│   ├── UserController.java
│   ├── UserService.java
│   ├── UserRepository.java
│   └── dto/
├── order/
│   ├── OrderController.java
│   └── ...
└── config/
```

### Orden de Elementos dentro de la Clase

1. Campos estáticos (constantes primero)
2. Campos de instancia
3. Constructores
4. Métodos públicos
5. Métodos protegidos
6. Métodos privados
7. Clases internas

### Formatting

- **Indentación**: 4 espacios (no tabs)
- **Línea máxima**: 120 caracteres
- **Braces**: estilo K&R (apertura en la misma línea)
- **Blank lines**: una entre métodos, dos entre secciones principales
- **Imports**: sin wildcards (`import java.util.List` no `import java.util.*`), organizados por grupo (java, javax, org, com, proyecto)

### Javadoc

Obligatorio en todas las clases públicas y métodos públicos:

```java
/**
 * Servicio para gestión de usuarios.
 * Maneja operaciones CRUD y validaciones de negocio.
 */
@Service
public class UserService {

    /**
     * Busca un usuario por su ID.
     *
     * @param id identificador único del usuario
     * @return el usuario encontrado
     * @throws UserNotFoundException si el usuario no existe
     */
    public UserDto findById(Long id) { ... }
}
```

## 2 - Security Patterns

### Validación de Entrada

```java
// ❌ Sin validación
@PostMapping("/users")
public ResponseEntity<User> create(@RequestBody UserDto dto) { ... }

// ✅ Con Bean Validation
@PostMapping("/users")
public ResponseEntity<User> create(@Valid @RequestBody UserDto dto) { ... }
```

Usar anotaciones de validación en DTOs:

```java
public record CreateUserDto(
    @NotBlank @Size(max = 100) String name,
    @Email @NotBlank String email,
    @NotNull @Min(0) Integer age
) {}
```

### SQL Injection

- SIEMPRE usar JPA/Hibernate con named parameters o Spring Data query methods
- NUNCA concatenar strings para queries

```java
// ❌ Vulnerable a SQL injection
@Query("SELECT u FROM User u WHERE u.name = '" + name + "'")

// ✅ Parametrizado
@Query("SELECT u FROM User u WHERE u.name = :name")
List<User> findByName(@Param("name") String name);
```

### Exposición de Datos

```java
// ❌ Expone la entidad completa (incluye password, datos internos)
@GetMapping("/users/{id}")
public User getUser(@PathVariable Long id) { return userRepository.findById(id); }

// ✅ Usa DTO para controlar qué se expone
@GetMapping("/users/{id}")
public UserResponseDto getUser(@PathVariable Long id) { return userService.findById(id); }
```

### Anti-patrones a Detectar

- Secretos hardcodeados en código → usar `application.yml` con variables de entorno
- `@CrossOrigin("*")` → configurar CORS restrictivo en `WebSecurityConfig`
- Endpoints sin autenticación que deberían tenerla → verificar SecurityFilterChain
- Logging de datos sensibles (passwords, tokens, PII) → sanitizar antes de loguear
- Uso de `@SuppressWarnings` sin justificación → cada supresión debe tener comentario
- Excepciones swallowed (catch vacío) → mínimo loguear el error

## 3 - Performance y Optimización

### N+1 Queries

```java
// ❌ N+1: una query por cada order del usuario
List<User> users = userRepository.findAll();
users.forEach(u -> u.getOrders().size()); // lazy load = N queries extra

// ✅ Fetch join en una sola query
@Query("SELECT u FROM User u LEFT JOIN FETCH u.orders")
List<User> findAllWithOrders();

// ✅ O usar @EntityGraph
@EntityGraph(attributePaths = {"orders"})
List<User> findAll();
```

### Uso Eficiente de Streams

```java
// ❌ Múltiples iteraciones
List<String> names = users.stream().map(User::getName).collect(toList());
List<String> filtered = names.stream().filter(n -> n.startsWith("A")).collect(toList());

// ✅ Una sola pipeline
List<String> result = users.stream()
    .map(User::getName)
    .filter(n -> n.startsWith("A"))
    .toList();
```

### Conexiones y Recursos

- Verificar que conexiones HTTP, DB, y file handles se cierran (try-with-resources)
- Pool de conexiones configurado apropiadamente (HikariCP defaults)
- Caching donde tenga sentido (`@Cacheable` para datos que no cambian frecuentemente)
- Paginación en endpoints que devuelven listas (`Pageable`)

### Otros Patrones

- `Optional` en lugar de null checks para retornos que pueden estar vacíos
- `StringBuilder` para concatenación en loops (no `+` repetido)
- Records para DTOs inmutables (Java 16+)
- Sealed classes para jerarquías cerradas (Java 17+)
- `var` para tipos obvios en variables locales (Java 10+)

## 4 - Test Quality

### Estructura Esperada

```
src/test/java/
├── unit/           # Tests unitarios con mocks
├── integration/    # Tests con contexto Spring (@SpringBootTest)
└── e2e/            # Tests end-to-end (si aplica)
```

### Cobertura por Tipo

| Capa | Tipo de test | Qué testear |
|------|-------------|-------------|
| Controller | `@WebMvcTest` | Validación de request/response, status codes, serialización |
| Service | Unit test con mocks | Lógica de negocio, edge cases, excepciones |
| Repository | `@DataJpaTest` | Queries custom, projections, constraints |
| Integration | `@SpringBootTest` | Flujos completos, interacción entre capas |

### Naming Convention

```java
// Patrón: should_resultado_when_condición
@Test
void should_returnUser_when_validIdProvided() { ... }

@Test
void should_throwNotFoundException_when_userDoesNotExist() { ... }

@Test
void should_return400_when_emailIsInvalid() { ... }
```

### Assertions

```java
// ❌ Assertions genéricas
assertTrue(result != null);
assertTrue(result.size() == 3);

// ✅ Assertions descriptivas (AssertJ)
assertThat(result).isNotNull();
assertThat(result).hasSize(3);
assertThat(result).extracting(User::getName).contains("Alice", "Bob");
```

### Anti-patrones en Tests

- Tests sin assertions → cada test DEBE verificar algo
- Tests que dependen del orden de ejecución → cada test debe ser independiente
- Mocking excesivo → si mockeas más de 3 dependencias, el diseño probablemente necesita refactoring
- Tests que prueban la implementación en vez del comportamiento → testear el **qué**, no el **cómo**
- `@SpringBootTest` para tests unitarios → excesivo, usar mocks
- Tests sin cleanup de datos → `@Transactional` en tests de integración para rollback automático

## 5 - Spring Boot Best Practices

### Inyección de Dependencias

```java
// ❌ Field injection (no testeable, oculta dependencias)
@Service
public class UserService {
    @Autowired
    private UserRepository userRepository;
}

// ✅ Constructor injection (explícito, testeable, inmutable)
@Service
public class UserService {
    private final UserRepository userRepository;

    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }
}

// ✅ Alternativa con Lombok
@Service
@RequiredArgsConstructor
public class UserService {
    private final UserRepository userRepository;
}
```

### Configuración

```java
// ❌ Valores hardcodeados
private static final int TIMEOUT = 5000;

// ✅ Externalizado en application.yml con @ConfigurationProperties
@ConfigurationProperties(prefix = "app.http")
public record HttpConfig(
    @DefaultValue("5000") int timeout,
    @DefaultValue("3") int maxRetries
) {}
```

### Manejo de Excepciones

```java
// ❌ Excepciones genéricas
throw new RuntimeException("User not found");

// ✅ Excepciones de dominio + @ControllerAdvice
throw new UserNotFoundException(id);

@RestControllerAdvice
public class GlobalExceptionHandler {
    @ExceptionHandler(UserNotFoundException.class)
    public ResponseEntity<ErrorResponse> handle(UserNotFoundException ex) {
        return ResponseEntity.status(404).body(new ErrorResponse(ex.getMessage()));
    }
}
```

### API Design

- Usar `ResponseEntity` para controlar status codes explícitamente
- DTOs separados para request y response (no reusar la misma clase)
- Versionado de API (`/api/v1/...`) si el proyecto lo requiere
- Documentación con OpenAPI/Swagger (`springdoc-openapi`)

### Profiles y Entornos

- `application.yml` para defaults, `application-dev.yml` / `application-prod.yml` para sobrescribir
- Secretos NUNCA en archivos de configuración comiteados → variables de entorno o vault
- `@Profile` para beans condicionales por entorno
- Actuator endpoints protegidos en producción
