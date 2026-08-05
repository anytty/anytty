package com.anytty.app.goclient

/**
 * ClientPlatformFailure 是 Android Client Engine 平台原语的稳定失败分类。
 * 它只在 Kotlin secure store 与 binding platform pump 之间传递，不拥有连接或重试策略。
 */
class ClientPlatformFailure(val code: String, message: String) : RuntimeException(message)
