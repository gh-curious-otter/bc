package com.bc.plugin.actions

import com.bc.plugin.services.BcService
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.DialogWrapper
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.components.JBTextArea
import java.awt.Dimension
import javax.swing.JComponent

/**
 * Action to view bc logs.
 */
class LogsAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val service = BcService.getInstance(project)

        val logs = service.getLogs(100)

        LogsDialog(logs).show()
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        e.presentation.isEnabledAndVisible = project != null &&
            BcService.getInstance(project).isWorkspace()
    }
}

class LogsDialog(private val logs: String) : DialogWrapper(true) {
    init {
        title = "bc Logs"
        init()
    }

    override fun createCenterPanel(): JComponent {
        val textArea = JBTextArea(logs).apply {
            isEditable = false
            lineWrap = true
            wrapStyleWord = true
        }

        return JBScrollPane(textArea).apply {
            preferredSize = Dimension(800, 500)
        }
    }
}
