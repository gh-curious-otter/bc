package com.bc.plugin.actions

import com.bc.plugin.services.BcService
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent

/**
 * Action to refresh bc status.
 */
class RefreshAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val service = BcService.getInstance(project)

        service.detectWorkspace()
        val status = service.getStatus()

        val message = if (status != null) {
            "Refreshed: ${status.workingCount}/${status.agentCount} agents working"
        } else {
            "bc status refreshed"
        }

        NotificationGroupManager.getInstance()
            .getNotificationGroup("bc.Notifications")
            .createNotification(message, NotificationType.INFORMATION)
            .notify(project)
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        e.presentation.isEnabledAndVisible = project != null
    }
}
