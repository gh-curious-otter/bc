package com.bc.plugin.services

import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.intellij.openapi.startup.ProjectActivity

/**
 * Startup activity to detect bc workspace and initialize plugin.
 */
class BcStartupActivity : ProjectActivity {
    private val logger = Logger.getInstance(BcStartupActivity::class.java)

    override suspend fun execute(project: Project) {
        val service = BcService.getInstance(project)

        if (service.detectWorkspace()) {
            logger.info("bc workspace detected in project: ${project.name}")
        }
    }
}
