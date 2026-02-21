package com.bc.plugin.settings

import com.bc.plugin.services.BcService
import com.intellij.openapi.options.Configurable
import com.intellij.openapi.project.Project
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.FormBuilder
import java.awt.BorderLayout
import javax.swing.JButton
import javax.swing.JComponent
import javax.swing.JPanel

/**
 * Settings panel for bc plugin configuration.
 */
class BcSettingsConfigurable(private val project: Project) : Configurable {
    private var bcPathField: JBTextField? = null
    private var panel: JPanel? = null

    override fun getDisplayName(): String = "bc"

    override fun createComponent(): JComponent {
        bcPathField = JBTextField()

        val testButton = JButton("Test Connection").apply {
            addActionListener { testConnection() }
        }

        val pathPanel = JPanel(BorderLayout()).apply {
            add(bcPathField!!, BorderLayout.CENTER)
            add(testButton, BorderLayout.EAST)
        }

        panel = FormBuilder.createFormBuilder()
            .addLabeledComponent(JBLabel("bc binary path:"), pathPanel)
            .addComponentFillVertically(JPanel(), 0)
            .panel

        return panel!!
    }

    override fun isModified(): Boolean {
        val service = BcService.getInstance(project)
        return bcPathField?.text != service.getBcPath()
    }

    override fun apply() {
        val service = BcService.getInstance(project)
        bcPathField?.text?.let { service.setBcPath(it) }
    }

    override fun reset() {
        val service = BcService.getInstance(project)
        bcPathField?.text = service.getBcPath()
    }

    private fun testConnection() {
        val service = BcService.getInstance(project)
        val path = bcPathField?.text ?: "bc"
        service.setBcPath(path)

        val status = service.getStatus()
        val message = if (status != null) {
            "Connection successful!\nWorkspace: ${status.workspace}\nAgents: ${status.agentCount}"
        } else {
            "Connection failed. Check bc path and ensure this is a bc workspace."
        }

        javax.swing.JOptionPane.showMessageDialog(
            panel,
            message,
            "bc Connection Test",
            if (status != null) javax.swing.JOptionPane.INFORMATION_MESSAGE
            else javax.swing.JOptionPane.ERROR_MESSAGE
        )
    }
}
