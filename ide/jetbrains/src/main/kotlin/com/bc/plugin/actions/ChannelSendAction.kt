package com.bc.plugin.actions

import com.bc.plugin.services.BcService
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages

/**
 * Action to send a message to a channel.
 */
class ChannelSendAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val service = BcService.getInstance(project)

        val channels = service.listChannels()
        if (channels.isEmpty()) {
            Messages.showWarningDialog(project, "No channels available.", "bc")
            return
        }

        val channel = Messages.showEditableChooseDialog(
            "Select channel:",
            "Send to Channel",
            Messages.getQuestionIcon(),
            channels.toTypedArray(),
            channels.first(),
            null
        ) ?: return

        val message = Messages.showInputDialog(
            project,
            "Message:",
            "Send to #$channel",
            null
        ) ?: return

        if (message.isBlank()) return

        val success = service.sendToChannel(channel, message)
        if (success) {
            Messages.showInfoMessage(project, "Message sent to #$channel", "bc")
        } else {
            Messages.showErrorDialog(project, "Failed to send message.", "bc")
        }
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        e.presentation.isEnabledAndVisible = project != null &&
            BcService.getInstance(project).isWorkspace()
    }
}
