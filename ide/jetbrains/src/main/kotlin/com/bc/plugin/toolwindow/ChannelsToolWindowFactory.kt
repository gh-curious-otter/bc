package com.bc.plugin.toolwindow

import com.bc.plugin.services.BcService
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.components.JBList
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.components.JBTextArea
import com.intellij.ui.content.ContentFactory
import java.awt.BorderLayout
import java.awt.Dimension
import javax.swing.*

/**
 * Tool window for bc channel communication.
 */
class ChannelsToolWindowFactory : ToolWindowFactory, DumbAware {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        val panel = ChannelsPanel(project)
        val content = ContentFactory.getInstance().createContent(panel, "Channels", false)
        toolWindow.contentManager.addContent(content)
    }

    override fun shouldBeAvailable(project: Project): Boolean {
        return BcService.getInstance(project).isWorkspace()
    }
}

class ChannelsPanel(private val project: Project) : JPanel(BorderLayout()) {
    private val channelList = JBList<String>()
    private val messagesArea = JBTextArea()
    private val messageInput = JTextField()
    private var selectedChannel: String? = null
    private val refreshTimer: Timer

    init {
        // Left panel - channel list
        val channelPanel = JPanel(BorderLayout()).apply {
            preferredSize = Dimension(150, 0)
            add(JLabel("Channels"), BorderLayout.NORTH)
            add(JBScrollPane(channelList), BorderLayout.CENTER)
        }

        channelList.addListSelectionListener {
            if (!it.valueIsAdjusting) {
                selectedChannel = channelList.selectedValue
                refreshMessages()
            }
        }

        // Right panel - messages
        messagesArea.apply {
            isEditable = false
            lineWrap = true
            wrapStyleWord = true
        }

        val inputPanel = JPanel(BorderLayout()).apply {
            add(messageInput, BorderLayout.CENTER)
            add(JButton("Send").apply {
                addActionListener { sendMessage() }
            }, BorderLayout.EAST)
        }

        messageInput.addActionListener { sendMessage() }

        val messagePanel = JPanel(BorderLayout()).apply {
            add(JBScrollPane(messagesArea), BorderLayout.CENTER)
            add(inputPanel, BorderLayout.SOUTH)
        }

        // Split pane
        val splitPane = JSplitPane(JSplitPane.HORIZONTAL_SPLIT, channelPanel, messagePanel).apply {
            dividerLocation = 150
        }

        add(splitPane, BorderLayout.CENTER)

        // Auto-refresh
        refreshTimer = Timer(5000) { refreshMessages() }
        refreshTimer.isRepeats = true
        refreshTimer.start()

        // Initial load
        refreshChannels()
    }

    private fun refreshChannels() {
        SwingUtilities.invokeLater {
            val service = BcService.getInstance(project)
            val channels = service.listChannels()
            channelList.setListData(channels.toTypedArray())
        }
    }

    private fun refreshMessages() {
        val channel = selectedChannel ?: return
        SwingUtilities.invokeLater {
            val service = BcService.getInstance(project)
            val messages = service.getChannelHistory(channel)

            messagesArea.text = messages.joinToString("\n") { msg ->
                "[${msg.timestamp}] ${msg.sender}: ${msg.message}"
            }

            // Scroll to bottom
            messagesArea.caretPosition = messagesArea.document.length
        }
    }

    private fun sendMessage() {
        val channel = selectedChannel ?: return
        val message = messageInput.text.trim()
        if (message.isEmpty()) return

        SwingUtilities.invokeLater {
            val service = BcService.getInstance(project)
            if (service.sendToChannel(channel, message)) {
                messageInput.text = ""
                refreshMessages()
            }
        }
    }
}
