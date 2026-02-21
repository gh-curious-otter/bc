package com.bc.plugin.actions

import com.bc.plugin.services.BcService
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent

/**
 * Action to show bc workspace status.
 */
class StatusAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val service = BcService.getInstance(project)

        val status = service.getStatus()
        val message = if (status != null) {
            """
            Workspace: ${status.workspace}
            Agents: ${status.agentCount}
            Active: ${status.activeCount}
            Working: ${status.workingCount}
            """.trimIndent()
        } else {
            "Unable to get bc status. Make sure bc is installed and this is a bc workspace."
        }

        NotificationGroupManager.getInstance()
            .getNotificationGroup("bc.Notifications")
            .createNotification("bc Status", message, NotificationType.INFORMATION)
            .notify(project)
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        e.presentation.isEnabledAndVisible = project != null &&
            BcService.getInstance(project).isWorkspace()
    }
}
