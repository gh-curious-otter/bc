package com.bc.plugin.actions

import com.bc.plugin.services.BcService
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages

/**
 * Action to list all channels.
 */
class ChannelListAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val service = BcService.getInstance(project)

        val channels = service.listChannels()
        val message = if (channels.isNotEmpty()) {
            channels.joinToString("\n") { "# $it" }
        } else {
            "No channels found."
        }

        Messages.showInfoMessage(project, message, "bc Channels")
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        e.presentation.isEnabledAndVisible = project != null &&
            BcService.getInstance(project).isWorkspace()
    }
}
