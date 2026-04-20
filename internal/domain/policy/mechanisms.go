package policy

// Имена механизмов защиты. Должны совпадать:
//   - с JSON-ключами в security_policies.mechanisms (JSONB);
//   - с Name() конкретных реализаций SecurityCheck;
//   - со значениями колонки security_check_results.mechanism.
//
// Строки — стабильный контракт; менять нельзя без миграции данных.
const (
	MechanismQRTTL = "qr_ttl"
	MechanismGeo   = "geo"
	MechanismWiFi  = "wifi"
)

// Reason — код причины для CheckResult.Details["reason"].
// Передаётся в UI преподавателя, чтобы ручное решение по skipped/failed
// было осознанным: одно дело «политика выключила механизм», другое —
// «клиент не прислал данные».
const (
	ReasonDisabled      = "disabled"       // механизм выключен в политике
	ReasonNoClientData  = "no_client_data" // клиент не прислал geo/bssid
	ReasonNoClassroom   = "no_classroom"   // у сессии нет привязки к аудитории (онлайн-формат)
	ReasonNoAllowed     = "no_allowed"     // allowlist пуст (некуда сравнивать)
	ReasonStale         = "stale"          // counter отстал больше чем на slack
	ReasonFutureCounter = "future_counter" // counter из будущего (подделка / рассинхронизация)
)
