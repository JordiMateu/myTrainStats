# myTrainStats 🚆

Agafo el tren per anar a treballar a Barcelona, i abans de sortir de casa sempre acabo mirant el mateix: com va l'R11, si el tren porta retard i si hi ha alguna afectació pel camí.

Vaig començar a mirar d'on treien les dades webs com La Vanguardia o Transporta'm i vaig acabar fent aquesta petita aplicació en Go. La idea és tenir el que em cal del trajecte **Girona ↔ Barcelona Sants** en una sola pantalla, des del terminal.

De moment està pensada per a aquest recorregut i per al meu dia a dia. Si també el fas, potser et serveix.

## Què hi puc veure?

- Un resum de l'**estat de l'R11**, amb els retards, les cancel·lacions i els avisos publicats per Renfe.
- Els trens del trajecte que apareixen a les dades en temps real: **AVE, Avant, Mitjana Distància i regionals**, segons la cobertura disponible.
- L'**hora de sortida programada** des de Girona o Sants, segons el sentit, i el retard publicat.
- Les **limitacions temporals de velocitat d'Adif** a l'àrea entre Barcelona i Girona.

Tot plegat amb [Bubble Tea](https://github.com/charmbracelet/bubbletea), Bubbles i Lip Gloss, perquè passar-se el dia al terminal no vol dir haver de mirar una paret de text.

## Com l'engego?

Cal tenir **Go 1.24 o posterior** i connexió a Internet. No cal cap clau d'API.

Des de la carpeta del projecte:

```sh
go run .
```

O bé compilar-la. A Windows:

```powershell
go build -o mytrainstats.exe .
.\mytrainstats.exe
```

A Linux o macOS:

```sh
go build -o mytrainstats .
./mytrainstats
```

## Tecles i opcions

| Tecla | Què fa |
| --- | --- |
| `↑` / `↓` | Moure's pel panell |
| `PgUp` / `PgDn` | Pujar o baixar una pàgina |
| `r` | Tornar a consultar les dades |
| `q` / `Ctrl+C` | Sortir |

Les consultes es fan en segon pla. Mentre s'actualitza, es continuen veient els resultats anteriors.

```sh
# Tornar a descarregar els horaris abans de consultar
go run . -refresh

# Fer una consulta en text, sense el panell interactiu
go run . -plain
```

Els horaris es desen a `.cache` durant 24 hores per no descarregar-los cada vegada. Les hores es mostren amb la zona horària `Europe/Madrid`. Si es redirigeix la sortida a un fitxer, s'utilitza automàticament el mode de text.

## D'on surten les dades?

De les fonts originals. Aquestes dues webs em van servir de punt de partida per investigar-ho:

- [La Vanguardia · Mobilitat](https://www.lavanguardia.com/movilidad/ca/), que explica les fonts a l'apartat «Com es fa».
- [Transporta'm · Fonts de dades](https://transportam.cat/data-sources).

L'aplicació consulta directament els horaris i els feeds de temps real de [Renfe Data](https://data.renfe.com/), i la capa pública del [visor GIS d'Adif de limitacions temporals de velocitat](https://experience.arcgis.com/experience/1d8dbf2ab3214b3c8d88592044ee13a8).

Per si vols remenar el codi o fer alguna cosa semblant:

| Dades | Font |
| --- | --- |
| Horaris AV/LD/MD | [GTFS](https://ssl.renfe.com/gtransit/Fichero_AV_LD/google_transit.zip) |
| Horaris de Cercanías i Rodalies | [GTFS](https://ssl.renfe.com/ftransit/Fichero_CER_FOMENTO/fomento_transit.zip) |
| Temps real AV/LD/MD | [Trip updates](https://gtfsrt.renfe.com/trip_updates_LD.json) |
| Temps real de Cercanías i Rodalies | [Trip updates](https://gtfsrt.renfe.com/trip_updates.json) |
| Avisos de Renfe | [Alerts](https://gtfsrt.renfe.com/alerts.json) |
| Limitacions de velocitat | [Capa ArcGIS d'Adif](https://services7.arcgis.com/XTupIrLX53AjaJqO/arcgis/rest/services/LTV_2/FeatureServer/0) |

Origen de les dades: **Renfe Operadora**, CC BY 4.0.

**© Administrador de infraestructuras ferroviarias**, per les dades d'Adif.

## Coses que cal tenir presents

La més important: **si no hi ha dades, no vol dir que el tren vagi a l'hora**. L'aplicació mostra el que rep i avisa si una font falla o si el feed de temps real té més de 15 minuts. La data del feed tampoc garanteix que cada tren s'hagi actualitzat fa poc.

El resum de l'R11 és un càlcul fet amb els trajectes que apareixen al feed de tota la línia. No és un estat oficial i pot incloure serveis d'altres hores o que no facin tot el tram Sants–Girona. Els avisos poden afectar altres punts de la línia.

La llista de trens tampoc és un horari complet de totes les sortides: només hi apareixen els que tenen una actualització en temps real i es poden relacionar amb els horaris. No inclou Iryo ni Ouigo. La sortida programada és la de l'horari, no una estimació; el retard es mostra a destinació quan existeix i, si només hi ha el retard general del tren, s'indica així. Només es marca una cancel·lació quan Renfe la publica explícitament.

Pel que fa a Adif, es consulten les limitacions geolocalitzades de les línies **050 i 270** dins el rectangle `2.05,41.35–2.9,42.05` en WGS84. És una aproximació geogràfica: pot incloure trams fora del trajecte i deixa fora les capes de càrrega o sense localització. No s'avaluen els calendaris, els horaris particulars ni si una restricció afecta un tren concret. Una limitació de velocitat no es tradueix directament en minuts de retard, i aquesta capa no recull totes les incidències.

Si una font falla, es mostra la resta amb un avís de consulta parcial. En mode de text, el programa acaba amb codi 1.

## Per tocar el codi

```sh
go test ./...
go vet ./...
```

És un projecte petit que aniré ajustant segons el que em trobi anant i tornant de la feina. Si hi veus alguna cosa que falla o tens una idea que faria més fàcil consultar els trens, obre una issue.
