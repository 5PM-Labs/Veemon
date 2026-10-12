# apps/mobile — Flutter client. Mirrors apps/api's make targets.
FLUTTER ?= flutter
DART    ?= dart
API_BASE_URL ?= http://10.0.2.2:3000

.PHONY: get gen watch analyze format format-check test test-coverage check run build-apk build-ios clean

get:            ## Install dependencies
	$(FLUTTER) pub get

gen: get        ## Generate freezed/json/injectable/mocks + l10n (never edit generated files)
	$(FLUTTER) gen-l10n
	$(DART) run build_runner build --delete-conflicting-outputs

watch:          ## Regenerate on change
	$(DART) run build_runner watch --delete-conflicting-outputs

analyze:        ## Must report zero issues
	$(FLUTTER) analyze

format:
	$(DART) format lib test

format-check:
	$(DART) format --output=none --set-exit-if-changed lib test

test:
	$(FLUTTER) test

test-coverage:
	$(FLUTTER) test --coverage

check: gen format-check analyze test  ## What CI runs

run:            ## Run against the local Go API (Android emulator host alias by default)
	$(FLUTTER) run --dart-define=API_BASE_URL=$(API_BASE_URL)

build-apk:      ## Compile check (debug)
	$(FLUTTER) build apk --debug --dart-define=API_BASE_URL=$(API_BASE_URL)

build-ios:      ## Compile check (debug, no signing)
	$(FLUTTER) build ios --debug --no-codesign --dart-define=API_BASE_URL=$(API_BASE_URL)

clean:
	$(FLUTTER) clean
