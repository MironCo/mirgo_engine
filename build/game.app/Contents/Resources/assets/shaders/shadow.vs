#version 330

// Minimal vertex shader for shadow pass - we only care about depth
in vec3 vertexPosition;

uniform mat4 mvp;

void main()
{
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}