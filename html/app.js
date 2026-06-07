class TurtleSoupGame {
    constructor() {
        this.ws = null;
        this.username = '';
        this.roomId = '';
        this.isAIResponding = false;
        this.aiResponseTimer = null;
        this.currentSoup = '';
        
        this.initElements();
        this.bindEvents();
    }

    initElements() {
        this.loginScreen = document.getElementById('login-screen');
        this.gameScreen = document.getElementById('game-screen');
        this.usernameInput = document.getElementById('username');
        this.roomIdInput = document.getElementById('room-id');
        this.joinBtn = document.getElementById('join-btn');
        this.backBtn = document.getElementById('back-btn');
        this.messageInput = document.getElementById('message-input');
        this.sendBtn = document.getElementById('send-btn');
        this.askBtn = document.getElementById('ask-btn');
        this.newPuzzleBtn = document.getElementById('new-puzzle-btn');
        this.messageList = document.getElementById('message-list');
        this.userList = document.getElementById('user-list');
        this.currentRoom = document.getElementById('current-room');
        this.onlineCount = document.getElementById('online-count');
        this.puzzleBackground = document.getElementById('puzzle-background');
    }

    bindEvents() {
        this.joinBtn.addEventListener('click', () => this.joinGame());
        this.backBtn.addEventListener('click', () => this.leaveGame());
        this.sendBtn.addEventListener('click', () => this.sendChat());
        this.askBtn.addEventListener('click', () => this.askAI());
        this.newPuzzleBtn.addEventListener('click', () => this.newPuzzle());
        this.messageInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendChat();
            }
        });
    }

    joinGame() {
        const username = this.usernameInput.value.trim();
        const roomId = this.roomIdInput.value.trim() || this.generateRoomId();

        if (!username) {
            alert('请输入昵称');
            return;
        }

        this.username = username;
        this.roomId = roomId;

        this.connectWebSocket();
        
        this.loginScreen.classList.remove('active');
        this.gameScreen.classList.add('active');
        this.currentRoom.textContent = roomId;
    }

    generateRoomId() {
        return 'room_' + Math.random().toString(36).substring(2, 10);
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${window.location.host}/ws?id=${this.roomId}&username=${encodeURIComponent(this.username)}`;

        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.addSystemMessage('已连接到房间');
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleMessage(message);
            } catch (e) {
                console.error('Parse message error:', e);
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.addSystemMessage('连接出现错误');
        };

        this.ws.onclose = () => {
            console.log('WebSocket closed');
            this.addSystemMessage('连接已断开');
            setTimeout(() => {
                this.leaveGame();
            }, 3000);
        };
    }

    handleMessage(message) {
        console.log('Received message:', message);
        switch (message.type) {
            case 'chat':
                this.addChatMessage(message.role, message.content);
                break;
            case 'answer':
                this.addAIMessage(message.content);
                break;
            case 'init':
                this.updateSoupContent(message.content);
                this.addAIMessage(message.content);
                break;
            case 'typing_start':
                this.showLoading(true);
                break;
            case 'typing_end':
                this.showLoading(false);
                break;
            case 'userlist':
                this.updateUserList(message.content);
                break;
            case 'soup':
                this.currentSoup = message.content;
                this.puzzleBackground.textContent = message.content || '等待主持人开始游戏...';
                break;
            default:
                console.log('Unknown message type:', message.type);
        }
    }
    
    updateSoupContent(content) {
        console.log('updateSoupContent called with:', content);
        if (!this.puzzleBackground) {
            console.error('puzzleBackground element not found!');
            return;
        }
        this.currentSoup += content;
        this.puzzleBackground.textContent = this.currentSoup;
        console.log('Soup content updated:', this.currentSoup);
    }

    showLoading(isLoading) {
        if (isLoading) {
            this.showTypingIndicator();
            this.askBtn.disabled = true;
            this.newPuzzleBtn.disabled = true;
            this.sendBtn.disabled = true;
            this.messageInput.disabled = true;
            
            this.askBtn.style.opacity = '0.5';
            this.newPuzzleBtn.style.opacity = '0.5';
            this.sendBtn.style.opacity = '0.5';
            this.messageInput.style.opacity = '0.5';
        } else {
            const typingIndicator = this.messageList.querySelector('.typing-indicator');
            if (typingIndicator) {
                typingIndicator.remove();
            }
            this.askBtn.disabled = false;
            this.newPuzzleBtn.disabled = false;
            this.sendBtn.disabled = false;
            this.messageInput.disabled = false;
            
            this.askBtn.style.opacity = '1';
            this.newPuzzleBtn.style.opacity = '1';
            this.sendBtn.style.opacity = '1';
            this.messageInput.style.opacity = '1';
        }
    }

    addChatMessage(sender, content) {
        const isOwn = sender === this.username;
        
        const messageItem = document.createElement('div');
        messageItem.className = `message-item ${isOwn ? 'own' : 'other'}`;
        
        messageItem.innerHTML = `
            <div class="message-header">
                <div class="message-avatar">${sender.charAt(0).toUpperCase()}</div>
                <span class="message-sender">${sender}</span>
                <span class="message-time">${this.getCurrentTime()}</span>
            </div>
            <div class="message-content">${this.escapeHtml(content)}</div>
        `;
        
        this.messageList.appendChild(messageItem);
        this.scrollToBottom();
    }

    addAIMessage(content) {
        const existingAIItem = this.messageList.querySelector('.message-item.ai:last-child');
        
        if (existingAIItem) {
            const hasTimestamp = existingAIItem.querySelector('.message-time');
            const contentElement = existingAIItem.querySelector('.message-content');
            
            if (!hasTimestamp && contentElement) {
                contentElement.textContent += content;
                this.scrollToBottom();
                this.resetAITimer();
                return;
            }
        }

        const typingIndicator = this.messageList.querySelector('.typing-indicator');
        if (typingIndicator) {
            typingIndicator.remove();
        }

        const messageItem = document.createElement('div');
        messageItem.className = 'message-item ai';
        
        messageItem.innerHTML = `
            <div class="message-header">
                <div class="message-avatar">🤖</div>
                <span class="message-sender">海龟汤AI</span>
            </div>
            <div class="message-content">${this.escapeHtml(content)}</div>
        `;
        
        this.messageList.appendChild(messageItem);
        this.scrollToBottom();
        this.resetAITimer();
    }

    resetAITimer() {
        if (this.aiResponseTimer) {
            clearTimeout(this.aiResponseTimer);
        }
        
        this.aiResponseTimer = setTimeout(() => {
            const lastAIItem = this.messageList.querySelector('.message-item.ai:last-child');
            if (lastAIItem) {
                const timeElement = lastAIItem.querySelector('.message-time');
                if (!timeElement) {
                    const header = lastAIItem.querySelector('.message-header');
                    if (header) {
                        header.innerHTML += `<span class="message-time">${this.getCurrentTime()}</span>`;
                    }
                }
            }
            this.isAIResponding = false;
        }, 500);
    }

    addSystemMessage(text) {
        const systemMessage = document.createElement('div');
        systemMessage.className = 'system-message';
        systemMessage.innerHTML = `<span>${text}</span>`;
        this.messageList.appendChild(systemMessage);
        this.scrollToBottom();
    }

    showTypingIndicator() {
        if (this.messageList.querySelector('.typing-indicator')) {
            return;
        }

        const typingIndicator = document.createElement('div');
        typingIndicator.className = 'typing-indicator';
        typingIndicator.innerHTML = `
            <div class="typing-dot"></div>
            <div class="typing-dot"></div>
            <div class="typing-dot"></div>
        `;
        
        const messageItem = document.createElement('div');
        messageItem.className = 'message-item ai';
        messageItem.innerHTML = `
            <div class="message-header">
                <div class="message-avatar">🤖</div>
                <span class="message-sender">海龟汤AI</span>
            </div>
        `;
        messageItem.appendChild(typingIndicator);
        
        this.messageList.appendChild(messageItem);
        this.scrollToBottom();
    }

    sendChat() {
        const content = this.messageInput.value.trim();
        if (!content || !this.ws || this.ws.readyState !== WebSocket.OPEN) {
            return;
        }

        const message = {
            role: this.username,
            type: 'chat',
            content: content
        };

        this.ws.send(JSON.stringify(message));
        this.messageInput.value = '';
    }

    askAI() {
        const content = this.messageInput.value.trim();
        if (!content || !this.ws || this.ws.readyState !== WebSocket.OPEN) {
            alert('请先输入问题内容');
            return;
        }

        const chatMessage = {
            role: this.username,
            type: 'chat',
            content: content
        };
        this.ws.send(JSON.stringify(chatMessage));

        const questionMessage = {
            role: this.username,
            type: 'question',
            content: content
        };
        this.ws.send(JSON.stringify(questionMessage));

        this.messageInput.value = '';
        this.showTypingIndicator();
    }

    newPuzzle() {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            alert('WebSocket 未连接');
            return;
        }

        this.currentSoup = '';
        this.puzzleBackground.textContent = '正在生成新谜题...';
        this.addSystemMessage('正在获取新谜题...');
        
        const message = {
            role: this.username,
            type: 'init',
            content: 'new puzzle'
        };

        console.log('发送 init 消息:', message);
        this.ws.send(JSON.stringify(message));
        this.showTypingIndicator();
    }

    updateUserList(userListStr) {
        const users = userListStr ? userListStr.split(', ') : [];
        this.onlineCount.textContent = `在线: ${users.length}`;
        
        this.userList.innerHTML = '';
        
        users.forEach(user => {
            const userItem = document.createElement('div');
            userItem.className = 'user-item online';
            userItem.textContent = user;
            this.userList.appendChild(userItem);
        });
        
        if (users.length === 0) {
            const emptyItem = document.createElement('div');
            emptyItem.className = 'user-item';
            emptyItem.textContent = '暂无在线玩家';
            this.userList.appendChild(emptyItem);
        }
    }

    leaveGame() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        
        this.gameScreen.classList.remove('active');
        this.loginScreen.classList.add('active');
        
        this.messageList.innerHTML = '<div class="system-message"><span>欢迎来到海龟汤游戏！</span></div>';
        this.userList.innerHTML = '<div class="user-item loading">加载中...</div>';
        this.puzzleBackground.textContent = '等待主持人开始游戏...';
        this.usernameInput.value = '';
        this.roomIdInput.value = '';
        this.username = '';
        this.roomId = '';
    }

    scrollToBottom() {
        this.messageList.scrollTop = this.messageList.scrollHeight;
    }

    getCurrentTime() {
        const now = new Date();
        return now.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new TurtleSoupGame();
});