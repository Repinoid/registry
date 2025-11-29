package controllers

import (
	"fmt"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
)

// (Версия 13.0: Чистый KeyValue Provider + Удаление всех DB настроек H2 и identity-providers.xml)
// generateNifiRegistryProperties генерирует содержимое nifi-registry.properties
func generateNifiRegistryProperties(nifiRegistry *registryv1.NifiRegistry, dbPassword string) string {
	var effectiveFlowProvider string
	var dbConfigSection string

	if nifiRegistry.Spec.Database.Enabled {
		// Конфигурация для внешней БД (PostgreSQL)
		effectiveFlowProvider = "org.apache.nifi.registry.flow.sql.SqlFlowProvider"
		dbConfigSection = fmt.Sprintf(`
# Database Configuration (PostgreSQL)
nifi.registry.db.implementation=org.apache.nifi.registry.db.sql.SqlFlowPersistenceProvider
nifi.registry.db.url=%s
nifi.registry.db.driver.class=%s
nifi.registry.db.username=%s
nifi.registry.db.password=%s`,
			nifiRegistry.Spec.Database.Url, nifiRegistry.Spec.Database.DriverClass,
			nifiRegistry.Spec.Database.Username, dbPassword)
	} else {
		// КОНФИГУРАЦИЯ ДЛЯ H2 (Встроенная БД)
		// Устанавливаем ТОЛЬКО KeyValue Provider. Полностью исключаем nifi.registry.db.*.
		effectiveFlowProvider = "org.apache.nifi.registry.flow.keyvalue.KeyValueFlowProvider"
		dbConfigSection = "" 
	}

	return fmt.Sprintf(`
# NiFi Registry Version
nifi.registry.version=1.24.0

# Flow Persistence Provider Settings
nifi.registry.flow.provider=%s
nifi.registry.flow.provider.implementation.org.apache.nifi.registry.flow.keyvalue.KeyValueFlowProvider.flow.storage.directory=./flow_storage
nifi.registry.flow.provider.implementation.org.apache.nifi.registry.flow.keyvalue.KeyValueFlowProvider.flow.storage.file=./flow_storage/flow.snapshot
nifi.registry.flow.provider.implementation.org.apache.nifi.registry.flow.keyvalue.KeyValueFlowProvider.flow.storage.max.entries=10000

%s

# Web Settings
nifi.registry.web.http.host=0.0.0.0
nifi.registry.web.http.port=8080
nifi.registry.web.context.path=/nifi-registry
nifi.registry.web.api.context.path=/nifi-registry-api
nifi.registry.web.jetty.threads=200

# Security Settings
# nifi.registry.security.user.login.identity.provider=keycloak 
nifi.registry.security.user.login.identity.provider=

# Access Policy Provider Settings
nifi.registry.security.access.resource.provider=org.apache.nifi.registry.security.authorization.file.FileAccessPolicyProvider
nifi.registry.security.access.resource.provider.implementation.org.apache.nifi.registry.security.authorization.file.FileAccessPolicyProvider.authorizations.file=./conf/authorizations.xml
nifi.registry.security.access.resource.provider.implementation.org.apache.nifi.registry.security.authorization.file.FileAccessPolicyProvider.initial.admin=

# User Group Provider Settings
nifi.registry.security.user.group.provider=org.apache.nifi.registry.security.authorization.file.FileUserGroupProvider
nifi.registry.security.user.group.provider.implementation.org.apache.nifi.registry.security.authorization.file.FileUserGroupProvider.users.file=./conf/users.xml
nifi.registry.security.user.group.provider.implementation.org.apache.nifi.registry.security.authorization.file.FileUserGroupProvider.initial.admin.identity=

# Identity Providers Settings
# nifi.registry.security.identity.providers.file=./conf/identity-providers.xml <--- КОММЕНТИРУЕМ ЭТУ СТРОКУ
# nifi.registry.security.identity.providers.file=./conf/identity-providers.xml

# Notification Service Settings
nifi.registry.notification.services.file=./conf/notification-services.xml

# SCM Persistence Providers Settings
nifi.registry.scm.providers.file=./conf/providers.xml

# Registry Aliases Settings
nifi.registry.registry.aliases.file=./conf/registry-aliases.xml

# Extension Bundles Settings
nifi.registry.extension.bundles.directory=./extension_bundles`,
		effectiveFlowProvider, dbConfigSection)
}