#version 330

// Simple fullscreen quad vertex shader for post-processing

in vec3 vertexPosition;
in vec2 vertexTexCoord;

out vec2 fragTexCoord;

void main() {
    fragTexCoord = vertexTexCoord;
    gl_Position = vec4(vertexPosition, 1.0);
}
