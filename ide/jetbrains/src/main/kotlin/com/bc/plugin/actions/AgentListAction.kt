package com.bc.plugin.actions

import com.bc.plugin.services.BcService
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages

/**
 * Action to list all agents.
 */
class AgentListAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val service = BcService.getInstance(project)

        val agents = service.listAgents()
        val message = if (agents.isNotEmpty()) {
            agents.joinToString("\n") { agent ->
                "${agent.name} (${agent.role}): ${agent.state}"
            }
        } else {
            "No agents found."
        }

        Messages.showInfoMessage(project, message, "bc Agents")
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        e.presentation.isEnabledAndVisible = project != null &&
            BcService.getInstance(project).isWorkspace()
    }
}
