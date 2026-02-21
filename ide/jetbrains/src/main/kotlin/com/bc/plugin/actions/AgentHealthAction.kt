package com.bc.plugin.actions

import com.bc.plugin.services.BcService
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages

/**
 * Action to check agent health.
 */
class AgentHealthAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val service = BcService.getInstance(project)

        val output = service.executeAndGetOutput("agent", "health")
        val message = output ?: "Unable to check agent health."

        Messages.showInfoMessage(project, message, "bc Agent Health")
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        e.presentation.isEnabledAndVisible = project != null &&
            BcService.getInstance(project).isWorkspace()
    }
}
