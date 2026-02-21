package com.bc.plugin.services

import com.intellij.openapi.project.Project
import com.intellij.openapi.util.Disposer
import com.intellij.openapi.wm.StatusBar
import com.intellij.openapi.wm.StatusBarWidget
import com.intellij.openapi.wm.StatusBarWidgetFactory
import com.intellij.util.Consumer
import java.awt.event.MouseEvent
import javax.swing.Timer

/**
 * Status bar widget factory showing bc workspace status.
 */
class BcStatusWidgetFactory : StatusBarWidgetFactory {
    override fun getId(): String = "BcStatusWidget"

    override fun getDisplayName(): String = "bc Status"

    override fun isAvailable(project: Project): Boolean {
        return BcService.getInstance(project).isWorkspace()
    }

    override fun createWidget(project: Project): StatusBarWidget {
        return BcStatusWidget(project)
    }

    override fun disposeWidget(widget: StatusBarWidget) {
        Disposer.dispose(widget)
    }

    override fun canBeEnabledOn(statusBar: StatusBar): Boolean = true
}

class BcStatusWidget(private val project: Project) : StatusBarWidget, StatusBarWidget.TextPresentation {
    private var statusBar: StatusBar? = null
    private var statusText = "bc: ..."
    private val refreshTimer: Timer

    init {
        refreshTimer = Timer(5000) { updateStatus() }
        refreshTimer.isRepeats = true
        refreshTimer.start()
        updateStatus()
    }

    override fun ID(): String = "BcStatusWidget"

    override fun getPresentation(): StatusBarWidget.WidgetPresentation = this

    override fun install(statusBar: StatusBar) {
        this.statusBar = statusBar
    }

    override fun dispose() {
        refreshTimer.stop()
    }

    override fun getText(): String = statusText

    override fun getAlignment(): Float = 0f

    override fun getTooltipText(): String = "bc AI Agent Orchestration - Click to refresh"

    override fun getClickConsumer(): Consumer<MouseEvent>? {
        return Consumer { updateStatus() }
    }

    private fun updateStatus() {
        val service = BcService.getInstance(project)
        val status = service.getStatus()

        statusText = if (status != null) {
            "bc: ${status.workingCount}/${status.agentCount} working"
        } else if (service.isWorkspace()) {
            "bc: offline"
        } else {
            "bc: not a workspace"
        }

        statusBar?.updateWidget(ID())
    }
}
