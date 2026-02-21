package com.bc.plugin.toolwindow

import com.bc.plugin.services.BcService
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.content.ContentFactory
import com.intellij.ui.table.JBTable
import java.awt.BorderLayout
import javax.swing.*
import javax.swing.table.DefaultTableModel

/**
 * Tool window showing bc agents status.
 */
class AgentsToolWindowFactory : ToolWindowFactory, DumbAware {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        val panel = AgentsPanel(project)
        val content = ContentFactory.getInstance().createContent(panel, "Agents", false)
        toolWindow.contentManager.addContent(content)
    }

    override fun shouldBeAvailable(project: Project): Boolean {
        return BcService.getInstance(project).isWorkspace()
    }
}

class AgentsPanel(private val project: Project) : JPanel(BorderLayout()) {
    private val tableModel = DefaultTableModel(
        arrayOf("Agent", "Role", "State", "Uptime", "Task"),
        0
    )
    private val table = JBTable(tableModel)
    private val refreshTimer: Timer

    init {
        // Header with refresh button
        val toolbar = JPanel().apply {
            layout = BoxLayout(this, BoxLayout.X_AXIS)
            add(JLabel("bc Agents"))
            add(Box.createHorizontalGlue())
            add(JButton("Refresh").apply {
                addActionListener { refreshAgents() }
            })
        }

        // Configure table
        table.apply {
            setShowGrid(true)
            autoResizeMode = JTable.AUTO_RESIZE_LAST_COLUMN
            columnModel.getColumn(0).preferredWidth = 100
            columnModel.getColumn(1).preferredWidth = 80
            columnModel.getColumn(2).preferredWidth = 60
            columnModel.getColumn(3).preferredWidth = 80
            columnModel.getColumn(4).preferredWidth = 200
        }

        add(toolbar, BorderLayout.NORTH)
        add(JBScrollPane(table), BorderLayout.CENTER)

        // Auto-refresh every 10 seconds
        refreshTimer = Timer(10000) { refreshAgents() }
        refreshTimer.isRepeats = true
        refreshTimer.start()

        // Initial load
        refreshAgents()
    }

    private fun refreshAgents() {
        SwingUtilities.invokeLater {
            val service = BcService.getInstance(project)
            val agents = service.listAgents()

            tableModel.rowCount = 0
            agents.forEach { agent ->
                tableModel.addRow(arrayOf(
                    agent.name,
                    agent.role,
                    agent.state,
                    agent.uptime,
                    agent.task
                ))
            }
        }
    }
}
