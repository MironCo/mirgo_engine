#version 330

in vec3 fragPosition;
in vec2 fragTexCoord;
in vec4 fragColor;
in vec3 fragNormal;
in vec2 fragShadowTexCoord;
in float fragShadowDepth;

uniform sampler2D texture0;
uniform sampler2D shadowMap;
uniform vec4 colDiffuse;
uniform vec3 viewPos;
uniform vec4 ambient;
uniform vec3 lightDir;
uniform vec4 lightColor;

out vec4 finalColor;

float calculateShadow(vec3 normal, vec3 lightDirection)
{
    // Check if fragment is outside shadowmap bounds
    if (fragShadowTexCoord.x < 0.0 || fragShadowTexCoord.x > 1.0 ||
        fragShadowTexCoord.y < 0.0 || fragShadowTexCoord.y > 1.0) {
        return 1.0; // No shadow outside the shadowmap
    }

    // Slope-scaled bias: surfaces at steep angles to light need more bias
    float NdotL = dot(normal, lightDirection);
    float bias = max(0.004 * (1.0 - NdotL), 0.0005);

    // PCF: sample a 5x5 kernel around the shadow coordinate
    vec2 texelSize = 1.0 / textureSize(shadowMap, 0);
    float shadow = 0.0;
    for (int x = -2; x <= 2; x++) {
        for (int y = -2; y <= 2; y++) {
            float sampleDepth = texture(shadowMap, fragShadowTexCoord + vec2(x, y) * texelSize).r;
            shadow += (fragShadowDepth - bias > sampleDepth) ? 0.0 : 1.0;
        }
    }
    shadow /= 25.0;

    // Remap from [0,1] to [shadowIntensity,1]
    float shadowIntensity = 0.3;
    return shadowIntensity + shadow * (1.0 - shadowIntensity);
}

void main()
{
    vec3 normal = normalize(fragNormal);
    vec3 viewDir = normalize(viewPos - fragPosition);
    // lightDir is direction light points (e.g. down), we need direction TOWARD light
    vec3 lightDirection = normalize(-lightDir);

    // Calculate shadow factor (pass normal and light dir for slope-scaled bias)
    float shadow = calculateShadow(normal, lightDirection);

    // Diffuse lighting (wrap lighting for softer look)
    float NdotL = dot(normal, lightDirection);
    float diff = max(NdotL, 0.0);
    // Add slight wrap lighting for softer shadows on geometry
    float wrapDiff = max(NdotL * 0.5 + 0.5, 0.0);
    diff = mix(diff, wrapDiff, 0.3);

    // Specular (Blinn-Phong) - tighter, brighter highlights
    vec3 halfwayDir = normalize(lightDirection + viewDir);
    float NdotH = max(dot(normal, halfwayDir), 0.0);
    float spec = pow(NdotH, 64.0);

    // Fresnel effect - brighter edges
    float fresnel = pow(1.0 - max(dot(normal, viewDir), 0.0), 3.0);

    // Rim lighting
    float rim = fresnel * max(dot(normal, lightDirection), 0.0);

    // Combine lighting components - apply shadow to diffuse and specular
    vec3 diffuseLight = diff * lightColor.rgb * shadow;
    vec3 specularLight = spec * lightColor.rgb * 1.2 * shadow;
    vec3 rimLight = rim * lightColor.rgb * 0.5 * shadow;
    vec3 fresnelLight = fresnel * ambient.rgb * 0.3;

    // Final lighting (ambient is not affected by shadow)
    vec3 lighting = ambient.rgb + diffuseLight + specularLight + rimLight + fresnelLight;

    // Apply material color
    vec3 baseColor = colDiffuse.rgb * fragColor.rgb;
    vec3 result = lighting * baseColor;

    // Slight tone mapping to prevent over-bright
    result = result / (result + vec3(1.0));

    // Gamma correction
    result = pow(result, vec3(1.0/2.2));

    finalColor = vec4(result, colDiffuse.a);
}
