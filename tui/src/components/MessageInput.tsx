import React, { useState, useCallback } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';

interface MessageInputProps {
  /** Placeholder text when input is empty */
  placeholder?: string;
  /** Called when message is submitted (Enter pressed) */
  onSubmit?: (message: string) => void;
  /** Called when input mode changes */
  onModeChange?: (isInputMode: boolean) => void;
  /** Whether input is disabled */
  disabled?: boolean;
  /** Channel name for display */
  channelName?: string;
}

/**
 * Message input component with keyboard mode toggle.
 *
 * Modes:
 * - Navigation mode (default): j/k navigation, i to enter input mode
 * - Input mode: Type message, Enter to submit, Escape to exit
 */
export const MessageInput: React.FC<MessageInputProps> = ({
  placeholder = 'Type a message...',
  onSubmit,
  onModeChange,
  disabled = false,
  channelName,
}) => {
  const [value, setValue] = useState('');
  const [isInputMode, setIsInputMode] = useState(false);

  const enterInputMode = useCallback(() => {
    if (!disabled) {
      setIsInputMode(true);
      onModeChange?.(true);
    }
  }, [disabled, onModeChange]);

  const exitInputMode = useCallback(() => {
    setIsInputMode(false);
    onModeChange?.(false);
  }, [onModeChange]);

  const handleSubmit = useCallback((text: string) => {
    if (text.trim()) {
      onSubmit?.(text.trim());
      setValue('');
    }
    // Stay in input mode after submit for quick follow-up messages
  }, [onSubmit]);

  // Handle keyboard input based on mode
  useInput((input, key) => {
    if (isInputMode) {
      // In input mode, Escape exits
      if (key.escape) {
        exitInputMode();
      }
    } else {
      // In navigation mode, 'i' or Enter enters input mode
      if (input === 'i' || key.return) {
        enterInputMode();
      }
    }
  }, { isActive: !disabled });

  if (disabled) {
    return (
      <Box borderStyle="single" borderColor="gray" paddingX={1}>
        <Text color="gray">Input disabled</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      {/* Input area */}
      <Box borderStyle="single" borderColor={isInputMode ? 'green' : 'gray'} paddingX={1}>
        {isInputMode ? (
          <Box>
            <Text color="green">&gt; </Text>
            <TextInput
              value={value}
              onChange={setValue}
              onSubmit={handleSubmit}
              placeholder={placeholder}
            />
          </Box>
        ) : (
          <Text color="gray">
            Press 'i' to type a message
            {channelName && <Text> to #{channelName}</Text>}
          </Text>
        )}
      </Box>

      {/* Mode indicator */}
      <Box>
        <Text color="gray" dimColor>
          {isInputMode
            ? 'Type message, Enter to send, Escape to exit'
            : 'i: input mode | j/k: scroll'}
        </Text>
      </Box>
    </Box>
  );
};

export default MessageInput;
