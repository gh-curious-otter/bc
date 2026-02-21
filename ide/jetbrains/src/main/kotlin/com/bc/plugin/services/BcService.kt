package com.bc.plugin.services

import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.execution.process.OSProcessHandler
import com.intellij.execution.process.ProcessOutput
import com.intellij.execution.util.ExecUtil
import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import java.io.File

/**
 * Service for interacting with the bc CLI.
 */
@Service(Service.Level.PROJECT)
class BcService(private val project: Project) {
    private val logger = Logger.getInstance(BcService::class.java)

    private var bcPath: String = "bc"
    private var isWorkspace: Boolean = false

    init {
        detectWorkspace()
    }

    /**
     * Check if the project is a bc workspace (has .bc directory)
     */
    fun detectWorkspace(): Boolean {
        val basePath = project.basePath ?: return false
        val bcDir = File(basePath, ".bc")
        isWorkspace = bcDir.exists() && bcDir.isDirectory
        return isWorkspace
    }

    fun isWorkspace(): Boolean = isWorkspace

    /**
     * Execute a bc command and return the output
     */
    fun execute(vararg args: String): ProcessOutput? {
        if (!isWorkspace) return null

        return try {
            val commandLine = GeneralCommandLine(bcPath)
                .withParameters(*args)
                .withWorkDirectory(project.basePath)
                .withEnvironment("NO_COLOR", "1")

            ExecUtil.execAndGetOutput(commandLine, 30000)
        } catch (e: Exception) {
            logger.warn("Failed to execute bc command: ${args.joinToString(" ")}", e)
            null
        }
    }

    /**
     * Execute bc command and return stdout
     */
    fun executeAndGetOutput(vararg args: String): String? {
        val output = execute(*args) ?: return null
        return if (output.exitCode == 0) output.stdout else null
    }

    /**
     * Get workspace status
     */
    fun getStatus(): BcStatus? {
        val output = executeAndGetOutput("status", "--json") ?: return null
        return parseStatus(output)
    }

    /**
     * List all agents
     */
    fun listAgents(): List<Agent> {
        val output = executeAndGetOutput("agent", "list", "--json") ?: return emptyList()
        return parseAgents(output)
    }

    /**
     * List all channels
     */
    fun listChannels(): List<String> {
        val output = executeAndGetOutput("channel", "list") ?: return emptyList()
        return output.lines()
            .filter { it.isNotBlank() }
            .map { it.trim() }
    }

    /**
     * Get channel history
     */
    fun getChannelHistory(channel: String, limit: Int = 20): List<ChannelMessage> {
        val output = executeAndGetOutput("channel", "history", channel, "--limit", limit.toString())
            ?: return emptyList()
        return parseChannelHistory(output)
    }

    /**
     * Send message to channel
     */
    fun sendToChannel(channel: String, message: String): Boolean {
        val output = execute("channel", "send", channel, message)
        return output?.exitCode == 0
    }

    /**
     * Get recent logs
     */
    fun getLogs(limit: Int = 50): String {
        return executeAndGetOutput("logs", "--tail", limit.toString()) ?: ""
    }

    fun setBcPath(path: String) {
        bcPath = path
    }

    fun getBcPath(): String = bcPath

    // Data classes
    data class BcStatus(
        val workspace: String,
        val agentCount: Int,
        val activeCount: Int,
        val workingCount: Int
    )

    data class Agent(
        val name: String,
        val role: String,
        val state: String,
        val uptime: String,
        val task: String
    )

    data class ChannelMessage(
        val timestamp: String,
        val sender: String,
        val message: String
    )

    // Parsers
    private fun parseStatus(json: String): BcStatus? {
        // Simple parsing - in production would use kotlinx.serialization
        return try {
            val workspace = Regex(""""workspace":\s*"([^"]+)"""").find(json)?.groupValues?.get(1) ?: "unknown"
            val agentCount = Regex(""""agent_count":\s*(\d+)""").find(json)?.groupValues?.get(1)?.toInt() ?: 0
            val activeCount = Regex(""""active_count":\s*(\d+)""").find(json)?.groupValues?.get(1)?.toInt() ?: 0
            val workingCount = Regex(""""working_count":\s*(\d+)""").find(json)?.groupValues?.get(1)?.toInt() ?: 0
            BcStatus(workspace, agentCount, activeCount, workingCount)
        } catch (e: Exception) {
            null
        }
    }

    private fun parseAgents(json: String): List<Agent> {
        // Simple line-based parsing for table output
        return json.lines()
            .filter { it.contains("engineer") || it.contains("manager") || it.contains("root") }
            .mapNotNull { line ->
                val parts = line.split(Regex("\\s{2,}")).map { it.trim() }
                if (parts.size >= 4) {
                    Agent(
                        name = parts[0],
                        role = parts[1],
                        state = parts[2],
                        uptime = parts.getOrElse(3) { "-" },
                        task = parts.getOrElse(4) { "" }
                    )
                } else null
            }
    }

    private fun parseChannelHistory(output: String): List<ChannelMessage> {
        return output.lines()
            .filter { it.contains("[") && it.contains("]") }
            .mapNotNull { line ->
                val match = Regex("""\[(\d+)\]\s*\[([^\]]+)\]\s*([^:]+):\s*(.*)""").find(line)
                if (match != null) {
                    ChannelMessage(
                        timestamp = match.groupValues[2],
                        sender = match.groupValues[3].trim(),
                        message = match.groupValues[4].trim()
                    )
                } else null
            }
    }

    companion object {
        fun getInstance(project: Project): BcService {
            return project.getService(BcService::class.java)
        }
    }
}
